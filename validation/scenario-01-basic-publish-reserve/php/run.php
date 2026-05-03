<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-basic-php';
$jobId = getenv('JOB_ID') ?: $queue . '-job-001';
$payload = [
    'kind' => 'basic-reserve',
    'source' => 'validation',
    'sdk' => 'php',
    'value' => 1,
];

$client = new Client(
    redisConnOpts: new RedisConnOpts(
        host: $redisHost,
        port: 6379,
    ),
    clientName: 'omniq-core-validation-php',
);

$invalidPublishRejected = false;

try {
    try {
        /** @phpstan-ignore-next-line */
        $client->publish(queue: $queue, payload: 'raw-string');
    } catch (Throwable) {
        $invalidPublishRejected = true;
    }

    $publishedJobId = $client->publish(
        queue: $queue,
        jobId: $jobId,
        payload: $payload,
        timeoutMs: 30000,
        maxAttempts: 3,
        backoffMs: 5000,
    );

    $reserve = $client->reserve(queue: $queue);

    $result = [
        'sdk' => 'php',
        'queue' => $queue,
        'invalid_publish_rejected' => $invalidPublishRejected,
        'job_id' => $publishedJobId,
        'reserve' => $reserve === null ? null : [
            'status' => $reserve->status,
            'job_id' => $reserve->jobId ?? null,
            'payload' => $reserve->payload ?? null,
            'attempt' => $reserve->attempt ?? null,
            'max_attempts' => $reserve->maxAttempts ?? null,
            'gid' => $reserve->gid ?? null,
            'lease_token_present' => trim($reserve->leaseToken ?? '') !== '',
        ],
    ];

    echo json_encode($result, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
