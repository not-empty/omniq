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

$queue = getenv('QUEUE') ?: 'validation-s17-php';
$baseNowMs = 1775340000000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-001', payload: ['kind' => 'grouped-ack-fail', 'slot' => 'alpha-1'], gid: 'alpha', groupLimit: 1, maxAttempts: 3, backoffMs: 5000, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-002', payload: ['kind' => 'grouped-ack-fail', 'slot' => 'alpha-2'], gid: 'alpha', groupLimit: 1, maxAttempts: 3, backoffMs: 5000, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-001', payload: ['kind' => 'grouped-ack-fail', 'slot' => 'beta-1'], gid: 'beta', groupLimit: 1, maxAttempts: 1, backoffMs: 5000, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-002', payload: ['kind' => 'grouped-ack-fail', 'slot' => 'beta-2'], gid: 'beta', groupLimit: 1, maxAttempts: 1, backoffMs: 5000, nowMsOverride: $baseNowMs + 4);

    $alphaFirst = reserveJob($client, $queue, $baseNowMs + 100);
    $betaFirst = reserveJob($client, $queue, $baseNowMs + 101);

    $alphaFail = $client->ackFail(queue: $queue, jobId: $alphaFirst->jobId, leaseToken: $alphaFirst->leaseToken, error: 'retryable grouped fail', nowMsOverride: $baseNowMs + 150);
    $betaFail = $client->ackFail(queue: $queue, jobId: $betaFirst->jobId, leaseToken: $betaFirst->leaseToken, error: 'terminal grouped fail', nowMsOverride: $baseNowMs + 151);

    $base = sprintf('{%s}', $queue);
    $alphaInflightAfterFail = (int) ($redis->get($base . ':g:alpha:inflight') ?: 0);
    $betaInflightAfterFail = (int) ($redis->get($base . ':g:beta:inflight') ?: 0);
    $alphaReadyAfterFail = $redis->zScore($base . ':groups:ready', 'alpha') !== false;
    $betaReadyAfterFail = $redis->zScore($base . ':groups:ready', 'beta') !== false;

    $nextOne = reserveJob($client, $queue, $baseNowMs + 152);
    $nextTwo = reserveJob($client, $queue, $baseNowMs + 153);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'alpha_fail_status' => $alphaFail[0],
        'beta_fail_status' => $betaFail[0],
        'alpha_inflight_after_fail' => $alphaInflightAfterFail,
        'beta_inflight_after_fail' => $betaInflightAfterFail,
        'alpha_ready_after_fail' => $alphaReadyAfterFail,
        'beta_ready_after_fail' => $betaReadyAfterFail,
        'next_job_ids' => [$nextOne->jobId, $nextTwo->jobId],
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
