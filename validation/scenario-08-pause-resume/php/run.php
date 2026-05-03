<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s08-php';
$firstJob = $queue . '-job-001';
$secondJob = $queue . '-job-002';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);

try {
    $client->publish(queue: $queue, jobId: $firstJob, payload: ['kind' => 'pause-resume', 'n' => 1]);
    $client->publish(queue: $queue, jobId: $secondJob, payload: ['kind' => 'pause-resume', 'n' => 2]);

    $pausedBefore = $client->isPaused($queue);
    $first = $client->reserve(queue: $queue);
    if ($first === null || $first->status !== 'JOB') {
        throw new RuntimeException('unexpected first reserve');
    }

    $client->pause($queue);
    $pausedAfterPause = $client->isPaused($queue);
    $pausedReserve = $client->reserve(queue: $queue);

    $client->resume($queue);
    $pausedAfterResume = $client->isPaused($queue);
    $second = $client->reserve(queue: $queue);
    if ($second === null || $second->status !== 'JOB') {
        throw new RuntimeException('unexpected second reserve');
    }

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'paused_before' => $pausedBefore,
        'paused_after_pause' => $pausedAfterPause,
        'paused_after_resume' => $pausedAfterResume,
        'paused_reserve_status' => $pausedReserve?->status,
        'first_reserved_job_id' => $first->jobId,
        'second_reserved_job_id' => $second->jobId,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
