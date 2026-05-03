<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\QueueMonitor;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$queue = getenv('QUEUE') ?: 'validation-s29-php';
$invalidNames = ['', ' bad', 'bad ', 'bad:name', '{manual-tag}', 'bad/name', 'bad\\name', 'bad name'];

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);

try {
    $validJobId = $client->publish(
        queue: $queue,
        jobId: $queue . '-job-001',
        payload: ['kind' => 'queue-name-validation', 'sdk' => 'php'],
    );

    $invalidResults = [];
    foreach ($invalidNames as $name) {
        $publishRejected = false;
        $statsRejected = false;

        try {
            $client->publish(queue: $name, payload: ['kind' => 'invalid']);
        } catch (Throwable) {
            $publishRejected = true;
        }

        try {
            $monitor->stats($name);
        } catch (Throwable) {
            $statsRejected = true;
        }

        $invalidResults[] = [
            'queue' => $name,
            'publish_rejected' => $publishRejected,
            'stats_rejected' => $statsRejected,
        ];
    }

    if ($validJobId === '') {
        throw new RuntimeException('valid queue did not publish a job id');
    }
    foreach ($invalidResults as $row) {
        if (!$row['publish_rejected'] || !$row['stats_rejected']) {
            throw new RuntimeException('invalid queue names were not rejected consistently');
        }
    }

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'valid_job_id' => $validJobId,
        'invalid_results' => $invalidResults,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
