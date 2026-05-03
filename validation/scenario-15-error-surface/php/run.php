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

function capture(callable $fn): string
{
    try {
        $fn();
        return 'NO_ERROR';
    } catch (Throwable $exception) {
        return $exception->getMessage();
    }
}

$queue = getenv('QUEUE') ?: 'validation-s15-php';
$baseNowMs = 1775320000000;

$jobId = $queue . '-job-001';
$delayedJob = $queue . '-delayed-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $invalidPublish = capture(
        static fn(): string => $client->publishJson(queue: $queue, jobId: $queue . '-bad-publish', payload: 'raw-string')
    );

    $client->publish(queue: $queue, jobId: $jobId, payload: ['kind' => 'error-surface'], nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $delayedJob, payload: ['kind' => 'error-surface', 'slot' => 'delayed'], dueMs: $baseNowMs + 100000, nowMsOverride: $baseNowMs + 2);

    $reserved = reserveJob($client, $queue, $baseNowMs + 100);

    $tokenMismatch = capture(static function () use ($client, $queue, $reserved, $baseNowMs): void {
        $client->ackSuccess(
            queue: $queue,
            jobId: $reserved->jobId,
            leaseToken: 'bad-token',
            nowMsOverride: $baseNowMs + 110,
        );
    });

    $redis->zRem(sprintf('{%s}:active', $queue), $reserved->jobId);

    $notActive = capture(static function () use ($client, $queue, $reserved, $baseNowMs): void {
        $client->ackSuccess(
            queue: $queue,
            jobId: $reserved->jobId,
            leaseToken: $reserved->leaseToken,
            nowMsOverride: $baseNowMs + 112,
        );
    });

    $batchLimit = capture(
        static fn(): array => $client->retryFailedBatch(
            queue: $queue,
            jobIds: array_map(
                static fn(int $i): string => sprintf('%s-x-%03d', $queue, $i),
                range(0, 100),
            ),
            nowMsOverride: $baseNowMs + 120,
        )
    );

    $laneMismatch = capture(
        static fn(): string => $client->removeJob(
            queue: $queue,
            jobId: $delayedJob,
            lane: 'wait',
        )
    );

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'token_mismatch' => $tokenMismatch,
        'not_active' => $notActive,
        'batch_limit' => $batchLimit,
        'invalid_publish' => $invalidPublish,
        'lane_mismatch' => $laneMismatch,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
