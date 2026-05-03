<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\QueueMonitor;
use Omniq\RedisConnOpts;
use Omniq\ReserveJob;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

function reserveJob(Client $client, string $queue, int $nowMs): ReserveJob
{
    $result = $client->reserve(queue: $queue, nowMsOverride: $nowMs);
    if (!$result instanceof ReserveJob || $result->status !== 'JOB') {
        throw new RuntimeException('unexpected reserve response');
    }

    return $result;
}

$queuePrefix = getenv('QUEUE') ?: 'validation-s31-php';
$queueA = $queuePrefix . '-a';
$queueB = $queuePrefix . '-b';
$baseNowMs = 1775450000000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    validation_script_flush($redis);
    $client->publish(
        queue: $queueA,
        jobId: $queueA . '-job-001',
        payload: ['kind' => 'multi-queue-noscript', 'queue' => 'a'],
        nowMsOverride: $baseNowMs + 1,
    );

    validation_script_flush($redis);
    $client->publish(
        queue: $queueB,
        jobId: $queueB . '-job-001',
        payload: ['kind' => 'multi-queue-noscript', 'queue' => 'b'],
        nowMsOverride: $baseNowMs + 2,
    );

    validation_script_flush($redis);
    $reservedA = reserveJob($client, $queueA, $baseNowMs + 100);

    validation_script_flush($redis);
    $client->ackSuccess(
        queue: $queueA,
        jobId: $reservedA->jobId,
        leaseToken: $reservedA->leaseToken,
        nowMsOverride: $baseNowMs + 110,
    );

    validation_script_flush($redis);
    $reservedB = reserveJob($client, $queueB, $baseNowMs + 120);

    validation_script_flush($redis);
    $heartbeatB = $client->heartbeat(
        queue: $queueB,
        jobId: $reservedB->jobId,
        leaseToken: $reservedB->leaseToken,
        nowMsOverride: $baseNowMs + 130,
    );

    validation_script_flush($redis);
    $client->ackSuccess(
        queue: $queueB,
        jobId: $reservedB->jobId,
        leaseToken: $reservedB->leaseToken,
        nowMsOverride: $baseNowMs + 140,
    );

    $queuesFound = array_values(array_filter(
        $monitor->scanQueues(),
        static fn(string $queue): bool => in_array($queue, [$queueA, $queueB], true),
    ));
    sort($queuesFound);
    $queueAState = (string) ($redis->hGet(sprintf('{%s}:job:%s', $queueA, $queueA . '-job-001'), 'state') ?: '');
    $queueBState = (string) ($redis->hGet(sprintf('{%s}:job:%s', $queueB, $queueB . '-job-001'), 'state') ?: '');

    if ($queuesFound !== [$queueA, $queueB]) {
        throw new RuntimeException('unexpected discovered queues');
    }
    if ($queueAState !== 'completed' || $queueBState !== 'completed') {
        throw new RuntimeException('multi-queue NOSCRIPT flow did not complete both jobs');
    }
    if ($heartbeatB <= 0) {
        throw new RuntimeException('heartbeat did not extend queue B lease');
    }

    echo json_encode([
        'sdk' => 'php',
        'queues_found' => $queuesFound,
        'queue_a_state' => $queueAState,
        'queue_b_state' => $queueBState,
        'heartbeat_b' => $heartbeatB,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
