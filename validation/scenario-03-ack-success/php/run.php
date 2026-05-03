<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s03-php';
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
            'kind' => 'ack-success',
            'source' => 'validation',
            'sdk' => 'php',
            'value' => 3,
        ],
        timeoutMs: 30000,
        maxAttempts: 3,
        backoffMs: 5000,
    );

    $reserve = $client->reserve(queue: $queue);
    if ($reserve === null || $reserve->status !== 'JOB') {
        throw new RuntimeException('unexpected reserve result');
    }

    $badError = '';
    try {
        $client->ackSuccess(queue: $queue, jobId: $reserve->jobId, leaseToken: 'bad-token');
    } catch (Throwable $exception) {
        $badError = $exception->getMessage();
    }

    $client->ackSuccess(queue: $queue, jobId: $reserve->jobId, leaseToken: $reserve->leaseToken);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'job_id' => $reserve->jobId,
        'ack_success_ok' => true,
        'invalid_token_error' => $badError,
        'invalid_token_contains_token_mismatch' => str_contains($badError, 'TOKEN_MISMATCH'),
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
