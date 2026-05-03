<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s07-php';
$retryJobId = getenv('RETRY_JOB_ID') ?: $queue . '-retry-job-001';
$failJobId = getenv('FAIL_JOB_ID') ?: $queue . '-fail-job-001';
$baseNowMs = 1775260000000;
$reapNowMs = $baseNowMs + 31000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);

try {
    $client->publish(
        queue: $queue,
        jobId: $retryJobId,
        payload: ['kind' => 'reap-expired', 'mode' => 'retry', 'sdk' => 'php'],
        timeoutMs: 30000,
        maxAttempts: 3,
        backoffMs: 5000,
        nowMsOverride: $baseNowMs,
    );
    $client->publish(
        queue: $queue,
        jobId: $failJobId,
        payload: ['kind' => 'reap-expired', 'mode' => 'terminal', 'sdk' => 'php'],
        timeoutMs: 30000,
        maxAttempts: 1,
        backoffMs: 5000,
        nowMsOverride: $baseNowMs,
    );

    $r1 = $client->reserve(queue: $queue, nowMsOverride: $baseNowMs);
    $r2 = $client->reserve(queue: $queue, nowMsOverride: $baseNowMs);
    if ($r1 === null || $r2 === null) {
        throw new RuntimeException('expected two reserved jobs');
    }

    $reaped = $client->reapExpired(queue: $queue, maxReap: 1000, nowMsOverride: $reapNowMs);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'reaped_count' => $reaped,
        'retryable_job_id' => $retryJobId,
        'terminal_job_id' => $failJobId,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
