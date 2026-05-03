<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s05-php';
$jobId = getenv('JOB_ID') ?: $queue . '-job-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);

try {
    $client->publish(
        queue: $queue,
        jobId: $jobId,
        payload: [
            'kind' => 'ack-fail-terminal',
            'source' => 'validation',
            'sdk' => 'php',
            'value' => 5,
        ],
        timeoutMs: 30000,
        maxAttempts: 1,
        backoffMs: 5000,
    );

    $reserve = $client->reserve(queue: $queue);
    if ($reserve === null || $reserve->status !== 'JOB') {
        throw new RuntimeException('unexpected reserve result');
    }

    $badError = '';
    try {
        $client->ackFail(queue: $queue, jobId: $reserve->jobId, leaseToken: 'bad-token', error: 'boom-terminal');
    } catch (Throwable $exception) {
        $badError = $exception->getMessage();
    }

    [$status, $dueMs] = $client->ackFail(
        queue: $queue,
        jobId: $reserve->jobId,
        leaseToken: $reserve->leaseToken,
        error: 'boom-terminal',
    );

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'job_id' => $reserve->jobId,
        'ack_fail_status' => $status,
        'due_ms' => $dueMs,
        'invalid_token_error' => $badError,
        'invalid_token_contains_token_mismatch' => str_contains($badError, 'TOKEN_MISMATCH'),
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
