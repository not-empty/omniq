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

$queue = getenv('QUEUE') ?: 'validation-s19-php';
$baseNowMs = 1775360000000;
$dueMs = $baseNowMs + 5000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-001', payload: ['kind' => 'grouped-promote-delayed', 'slot' => 'alpha-1'], gid: 'alpha', groupLimit: 1, dueMs: $dueMs, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-001', payload: ['kind' => 'grouped-promote-delayed', 'slot' => 'beta-1'], gid: 'beta', groupLimit: 1, dueMs: $dueMs, nowMsOverride: $baseNowMs + 2);

    $promotedCount = $client->promoteDelayed(queue: $queue, maxPromote: 1000, nowMsOverride: $dueMs);

    $base = sprintf('{%s}', $queue);
    $alphaReadyAfterPromote = $redis->zScore($base . ':groups:ready', 'alpha') !== false;
    $betaReadyAfterPromote = $redis->zScore($base . ':groups:ready', 'beta') !== false;
    $statsRaw = $redis->hGetAll($base . ':stats') ?: [];
    $groupWaitingAfterPromote = (int) ($statsRaw['group_waiting'] ?? 0);

    $nextOne = reserveJob($client, $queue, $dueMs + 1);
    $nextTwo = reserveJob($client, $queue, $dueMs + 2);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'promoted_count' => $promotedCount,
        'alpha_ready_after_promote' => $alphaReadyAfterPromote,
        'beta_ready_after_promote' => $betaReadyAfterPromote,
        'group_waiting_after_promote' => $groupWaitingAfterPromote,
        'next_job_ids' => [$nextOne->jobId, $nextTwo->jobId],
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
