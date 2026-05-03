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

$queue = getenv('QUEUE') ?: 'validation-s18-php';
$baseNowMs = 1775350000000;
$reapNowMs = $baseNowMs + 31000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-001', payload: ['kind' => 'grouped-reap-expired', 'slot' => 'alpha-1'], gid: 'alpha', groupLimit: 1, maxAttempts: 3, timeoutMs: 30000, backoffMs: 5000, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-002', payload: ['kind' => 'grouped-reap-expired', 'slot' => 'alpha-2'], gid: 'alpha', groupLimit: 1, maxAttempts: 3, timeoutMs: 30000, backoffMs: 5000, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-001', payload: ['kind' => 'grouped-reap-expired', 'slot' => 'beta-1'], gid: 'beta', groupLimit: 1, maxAttempts: 1, timeoutMs: 30000, backoffMs: 5000, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-002', payload: ['kind' => 'grouped-reap-expired', 'slot' => 'beta-2'], gid: 'beta', groupLimit: 1, maxAttempts: 1, timeoutMs: 30000, backoffMs: 5000, nowMsOverride: $baseNowMs + 4);

    reserveJob($client, $queue, $baseNowMs + 100);
    reserveJob($client, $queue, $baseNowMs + 101);

    $reapedCount = $client->reapExpired(queue: $queue, maxReap: 1000, nowMsOverride: $reapNowMs);

    $base = sprintf('{%s}', $queue);
    $alphaInflightAfterReap = (int) ($redis->get($base . ':g:alpha:inflight') ?: 0);
    $betaInflightAfterReap = (int) ($redis->get($base . ':g:beta:inflight') ?: 0);
    $alphaReadyAfterReap = $redis->zScore($base . ':groups:ready', 'alpha') !== false;
    $betaReadyAfterReap = $redis->zScore($base . ':groups:ready', 'beta') !== false;

    $nextOne = reserveJob($client, $queue, $reapNowMs + 1);
    $nextTwo = reserveJob($client, $queue, $reapNowMs + 2);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'reaped_count' => $reapedCount,
        'alpha_inflight_after_reap' => $alphaInflightAfterReap,
        'beta_inflight_after_reap' => $betaInflightAfterReap,
        'alpha_ready_after_reap' => $alphaReadyAfterReap,
        'beta_ready_after_reap' => $betaReadyAfterReap,
        'next_job_ids' => [$nextOne->jobId, $nextTwo->jobId],
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
