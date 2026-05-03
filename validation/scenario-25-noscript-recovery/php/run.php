<?php

declare(strict_types=1);

use Omniq\Client;
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

$queue = getenv('QUEUE') ?: 'validation-s25-php';
$baseNowMs = 1775420000000;

$publishJob = $queue . '-job-001';
$delayedJob = $queue . '-delayed-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    validation_script_flush($redis);
    $publishedJobId = $client->publish(
        queue: $queue,
        jobId: $publishJob,
        payload: ['kind' => 'noscript-recovery', 'slot' => 'publish'],
        nowMsOverride: $baseNowMs + 1,
    );

    validation_script_flush($redis);
    $reserved = reserveJob($client, $queue, $baseNowMs + 100);

    validation_script_flush($redis);
    $heartbeatLockUntilMs = $client->heartbeat(
        queue: $queue,
        jobId: $reserved->jobId,
        leaseToken: $reserved->leaseToken,
        nowMsOverride: $baseNowMs + 110,
    );

    validation_script_flush($redis);
    $client->ackSuccess(
        queue: $queue,
        jobId: $reserved->jobId,
        leaseToken: $reserved->leaseToken,
        nowMsOverride: $baseNowMs + 120,
    );

    validation_script_flush($redis);
    $delayedJobId = $client->publish(
        queue: $queue,
        jobId: $delayedJob,
        payload: ['kind' => 'noscript-recovery', 'slot' => 'delayed'],
        dueMs: $baseNowMs + 500,
        nowMsOverride: $baseNowMs + 2,
    );

    validation_script_flush($redis);
    $promotedCount = $client->promoteDelayed(
        queue: $queue,
        maxPromote: 10,
        nowMsOverride: $baseNowMs + 600,
    );

    $completedState = (string) ($redis->hGet(sprintf('{%s}:job:%s', $queue, $publishJob), 'state') ?: '');
    $promotedState = (string) ($redis->hGet(sprintf('{%s}:job:%s', $queue, $delayedJob), 'state') ?: '');

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'published_job_id' => $publishedJobId,
        'reserved_job_id' => $reserved->jobId,
        'heartbeat_lock_until_ms' => $heartbeatLockUntilMs,
        'completed_state' => $completedState,
        'delayed_job_id' => $delayedJobId,
        'promoted_count' => $promotedCount,
        'promoted_state' => $promotedState,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
