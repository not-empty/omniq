<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\QueueMonitor;
use Omniq\RedisConnOpts;
use Omniq\ReserveJob;

require '/workspace/omniq-php/vendor/autoload.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';
$queue = getenv('QUEUE') ?: 'validation-s32-php';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);

try {
    $client->publish(
        queue: $queue,
        jobId: $queue . '-job-001',
        payload: ['kind' => 'transport-backend-smoke', 'backend' => $redisMode, 'sdk' => 'php'],
    );

    $reserved = $client->reserve(queue: $queue);
    if (!$reserved instanceof ReserveJob || $reserved->status !== 'JOB') {
        throw new RuntimeException('unexpected reserve response');
    }

    $queuesFound = array_values(array_filter(
        $monitor->scanQueues(),
        static fn(string $found): bool => $found === $queue,
    ));
    if ($queuesFound !== [$queue]) {
        throw new RuntimeException('unexpected discovered queues');
    }

    echo json_encode([
        'sdk' => 'php',
        'backend' => $redisMode,
        'queue' => $queue,
        'reserve_status' => $reserved->status,
        'reserved_job_id' => $reserved->jobId,
        'queues_found' => $queuesFound,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
