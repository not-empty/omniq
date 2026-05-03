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

$queue = getenv('QUEUE') ?: 'validation-s09-php';
$baseNowMs = 1775270000000;
$alphaFirst = $queue . '-alpha-job-001';
$alphaSecond = $queue . '-alpha-job-002';
$betaFirst = $queue . '-beta-job-001';
$ungrouped = $queue . '-ungrouped-job-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $alphaFirst, payload: ['kind' => 'grouped-jobs', 'slot' => 'alpha-1', 'sdk' => 'php'], gid: 'alpha', groupLimit: 1, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $alphaSecond, payload: ['kind' => 'grouped-jobs', 'slot' => 'alpha-2', 'sdk' => 'php'], gid: 'alpha', groupLimit: 5, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $betaFirst, payload: ['kind' => 'grouped-jobs', 'slot' => 'beta-1', 'sdk' => 'php'], gid: 'beta', groupLimit: 1, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $ungrouped, payload: ['kind' => 'grouped-jobs', 'slot' => 'ungrouped-1', 'sdk' => 'php'], nowMsOverride: $baseNowMs + 4);

    $first = reserveJob($client, $queue, $baseNowMs + 100);
    $second = reserveJob($client, $queue, $baseNowMs + 101);
    $third = reserveJob($client, $queue, $baseNowMs + 102);
    $fourth = $client->reserve(queue: $queue, nowMsOverride: $baseNowMs + 103);

    $client->ackSuccess(queue: $queue, jobId: $first->jobId, leaseToken: $first->leaseToken, nowMsOverride: $baseNowMs + 200);
    $fifth = reserveJob($client, $queue, $baseNowMs + 201);

    $groupLimitAlpha = $redis->get(sprintf('{%s}:g:alpha:limit', $queue));
    $fourthStatus = $fourth === null ? 'EMPTY' : $fourth->status;

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'group_limit_alpha' => $groupLimitAlpha === false ? '' : (string) $groupLimitAlpha,
        'reserve_order' => [
            ['job_id' => $first->jobId, 'gid' => $first->gid],
            ['job_id' => $second->jobId, 'gid' => $second->gid],
            ['job_id' => $third->jobId, 'gid' => $third->gid],
        ],
        'fourth_reserve_status' => $fourthStatus,
        'fifth_reserve_job_id' => $fifth->jobId,
        'fifth_reserve_gid' => $fifth->gid,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
