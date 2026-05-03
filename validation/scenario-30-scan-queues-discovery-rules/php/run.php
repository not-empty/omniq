<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\QueueMonitor;
use Omniq\QueueStats;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';
$prefix = getenv('PREFIX') ?: 'validation-s30-php';
$queueA = $prefix . '-alpha';
$queueB = $prefix . '.beta_2';
$pausedOnly = $prefix . '-paused-only';
$invalidColonKey = $prefix . '-bad:name:stats';
$invalidSpaceKey = sprintf('{%s bad}:stats', $prefix);

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $redis->hMSet(sprintf('{%s}:stats', $queueA), ['waiting' => '0']);
    $redis->hMSet(sprintf('{%s}:stats', $queueB), ['waiting' => '1']);
    $redis->set(sprintf('{%s}:paused', $pausedOnly), '1');
    $redis->hMSet($invalidColonKey, ['waiting' => '9']);
    $redis->hMSet($invalidSpaceKey, ['waiting' => '9']);

    $queuesFound = array_values(array_filter(
        $monitor->scanQueues(),
        static fn(string $queue): bool => str_starts_with($queue, $prefix),
    ));
    sort($queuesFound);

    $statsManyAuto = array_values(array_map(
        static fn(QueueStats $stats): string => $stats->queue,
        array_filter(
            $monitor->statsMany(),
            static fn(QueueStats $stats): bool => str_starts_with($stats->queue, $prefix),
        ),
    ));
    sort($statsManyAuto);
    $expected = [$queueA, $queueB];
    sort($expected);

    if ($queuesFound !== $expected) {
        throw new RuntimeException('unexpected discovered queues');
    }
    if ($statsManyAuto !== $expected) {
        throw new RuntimeException('unexpected statsMany() discovery');
    }
    if (in_array($pausedOnly, $queuesFound, true)) {
        throw new RuntimeException('paused-only queue should not be discovered');
    }
    if (count(array_filter($queuesFound, static fn(string $queue): bool => str_contains($queue, 'bad'))) > 0) {
        throw new RuntimeException('invalid sparse keys leaked into queue discovery');
    }

    echo json_encode([
        'sdk' => 'php',
        'queues_found' => $queuesFound,
        'stats_many_auto' => $statsManyAuto,
        'paused_only_discovered' => in_array($pausedOnly, $queuesFound, true),
        'invalid_discovered' => count(array_filter($queuesFound, static fn(string $queue): bool => str_contains($queue, 'bad'))) > 0,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
