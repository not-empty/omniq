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

function statsToArray(QueueStats $stats): array
{
    return [
        'queue' => $stats->queue,
        'paused' => $stats->paused,
        'waiting' => $stats->waiting,
        'group_waiting' => $stats->groupWaiting,
        'waiting_total' => $stats->waitingTotal,
        'active' => $stats->active,
        'delayed' => $stats->delayed,
        'failed' => $stats->failed,
        'completed_kept' => $stats->completedKept,
        'groups_ready' => $stats->groupsReady,
        'last_activity_ms' => $stats->lastActivityMs,
        'last_enqueue_ms' => $stats->lastEnqueueMs,
        'last_reserve_ms' => $stats->lastReserveMs,
        'last_finish_ms' => $stats->lastFinishMs,
    ];
}

$prefix = getenv('PREFIX') ?: 'validation-s24-php';
$queueEmpty = $prefix . '-empty';
$queuePartial = $prefix . '-partial';
$queuePaused = $prefix . '-paused';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $redis->hMSet(sprintf('{%s}:stats', $queueEmpty), [
        'waiting' => '0',
    ]);
    $redis->hMSet(sprintf('{%s}:stats', $queuePartial), [
        'waiting' => '2',
        'group_waiting' => '1',
        'active' => '3',
        'last_activity_ms' => '1775410000001',
    ]);
    $redis->set(sprintf('{%s}:paused', $queuePaused), '1');

    $queuesFound = array_values(array_filter(
        $monitor->scanQueues(),
        static fn(string $queue): bool => in_array($queue, [$queueEmpty, $queuePartial, $queuePaused], true),
    ));
    sort($queuesFound);

    $statsEmpty = statsToArray($monitor->stats($queueEmpty));
    $statsPartial = statsToArray($monitor->stats($queuePartial));
    $statsPaused = statsToArray($monitor->stats($queuePaused));
    $statsMany = array_map(static fn(QueueStats $stats): array => statsToArray($stats), $monitor->statsMany([$queueEmpty, $queuePartial, $queuePaused]));

    echo json_encode([
        'sdk' => 'php',
        'queues_found' => $queuesFound,
        'stats_empty' => $statsEmpty,
        'stats_partial' => $statsPartial,
        'stats_paused' => $statsPaused,
        'stats_many' => $statsMany,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
