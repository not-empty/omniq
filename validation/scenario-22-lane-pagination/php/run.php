<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\LaneJob;
use Omniq\QueueMonitor;
use Omniq\QueueStats;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

/** @param list<LaneJob> $rows
 *  @return list<string>
 */
function jobIds(array $rows): array
{
    return array_map(static fn(LaneJob $row): string => $row->jobId, $rows);
}

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

$queue = getenv('QUEUE') ?: 'validation-s22-php';
$baseNowMs = 1775390000000;

$waitJobs = array_map(static fn(int $i): string => sprintf('%s-wait-%03d', $queue, $i), range(1, 5));
$delayedJobs = array_map(static fn(int $i): string => sprintf('%s-delayed-%03d', $queue, $i), range(1, 5));

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    foreach ($waitJobs as $idx => $jobId) {
        $order = $idx + 1;
        $client->publish(
            queue: $queue,
            jobId: $jobId,
            payload: ['kind' => 'lane-pagination', 'lane' => 'wait', 'order' => $order],
            nowMsOverride: $baseNowMs + $order,
        );
    }

    foreach ($delayedJobs as $idx => $jobId) {
        $order = $idx + 1;
        $client->publish(
            queue: $queue,
            jobId: $jobId,
            payload: ['kind' => 'lane-pagination', 'lane' => 'delayed', 'order' => $order],
            dueMs: $baseNowMs + 100000 + $order,
            nowMsOverride: $baseNowMs + 100 + $order,
        );
    }

    $waitForwardPages = [
        jobIds($monitor->lanePage($queue, 'wait', offset: 0, limit: 2, reverse: false)),
        jobIds($monitor->lanePage($queue, 'wait', offset: 2, limit: 2, reverse: false)),
        jobIds($monitor->lanePage($queue, 'wait', offset: 4, limit: 2, reverse: false)),
    ];
    $waitReversePages = [
        jobIds($monitor->lanePage($queue, 'wait', offset: 0, limit: 2, reverse: true)),
        jobIds($monitor->lanePage($queue, 'wait', offset: 2, limit: 2, reverse: true)),
        jobIds($monitor->lanePage($queue, 'wait', offset: 4, limit: 2, reverse: true)),
    ];
    $delayedForwardPages = [
        jobIds($monitor->lanePage($queue, 'delayed', offset: 0, limit: 2, reverse: false)),
        jobIds($monitor->lanePage($queue, 'delayed', offset: 2, limit: 2, reverse: false)),
        jobIds($monitor->lanePage($queue, 'delayed', offset: 4, limit: 2, reverse: false)),
    ];
    $delayedReversePages = [
        jobIds($monitor->lanePage($queue, 'delayed', offset: 0, limit: 2, reverse: true)),
        jobIds($monitor->lanePage($queue, 'delayed', offset: 2, limit: 2, reverse: true)),
        jobIds($monitor->lanePage($queue, 'delayed', offset: 4, limit: 2, reverse: true)),
    ];

    $stats = statsToArray($monitor->stats($queue));
    $idxWaitRaw = array_map('strval', $redis->zRange(sprintf('{%s}:idx:wait', $queue), 0, -1) ?: []);
    $idxDelayedRaw = array_map('strval', $redis->zRange(sprintf('{%s}:idx:delayed', $queue), 0, -1) ?: []);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'stats' => $stats,
        'wait_forward_pages' => $waitForwardPages,
        'wait_reverse_pages' => $waitReversePages,
        'delayed_forward_pages' => $delayedForwardPages,
        'delayed_reverse_pages' => $delayedReversePages,
        'idx_wait_raw' => $idxWaitRaw,
        'idx_delayed_raw' => $idxDelayedRaw,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
