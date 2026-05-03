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

$queue = getenv('QUEUE') ?: 'validation-s16-php';
$baseNowMs = 1775330000000;
$gid = 'alpha';
$firstJob = $queue . '-alpha-job-001';
$secondJob = $queue . '-alpha-job-002';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $firstJob, payload: ['kind' => 'grouped-ack-success', 'slot' => 'first'], gid: $gid, groupLimit: 1, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $secondJob, payload: ['kind' => 'grouped-ack-success', 'slot' => 'second'], gid: $gid, groupLimit: 1, nowMsOverride: $baseNowMs + 2);

    $first = reserveJob($client, $queue, $baseNowMs + 100);
    $client->ackSuccess(queue: $queue, jobId: $first->jobId, leaseToken: $first->leaseToken, nowMsOverride: $baseNowMs + 150);

    $base = sprintf('{%s}', $queue);
    $groupReadyAfterAck = $redis->zScore($base . ':groups:ready', $gid) !== false;
    $inflightRaw = $redis->get($base . ':g:' . $gid . ':inflight');
    $groupInflightAfterAck = $inflightRaw === false ? 0 : (int) $inflightRaw;

    $second = reserveJob($client, $queue, $baseNowMs + 151);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'first_job_id' => $first->jobId,
        'second_job_id' => $second->jobId,
        'group_ready_after_ack' => $groupReadyAfterAck,
        'group_inflight_after_ack' => $groupInflightAfterAck,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
