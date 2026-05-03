<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s06-php';
$jobId = getenv('JOB_ID') ?: $queue . '-job-001';
$baseNowMs = 1775250000000;
$dueMs = $baseNowMs + 5000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);

try {
    $client->publish(
        queue: $queue,
        jobId: $jobId,
        payload: [
            'kind' => 'promote-delayed',
            'source' => 'validation',
            'sdk' => 'php',
            'value' => 6,
        ],
        timeoutMs: 30000,
        maxAttempts: 3,
        backoffMs: 5000,
        dueMs: $dueMs,
        nowMsOverride: $baseNowMs,
    );

    $promoted = $client->promoteDelayed(queue: $queue, maxPromote: 1000, nowMsOverride: $dueMs);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'job_id' => $jobId,
        'scheduled_due_ms' => $dueMs,
        'promoted_count' => $promoted,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
