<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\GroupReady;
use Omniq\GroupStatus;
use Omniq\QueueMonitor;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

function groupReadyToArray(GroupReady $item): array
{
    return [
        'gid' => $item->gid,
        'score_ms' => $item->scoreMs,
    ];
}

function groupStatusToArray(GroupStatus $item): array
{
    return [
        'gid' => $item->gid,
        'inflight' => $item->inflight,
        'limit' => $item->limit,
        'ready' => $item->ready,
        'waiting_count' => $item->waitingCount,
    ];
}

$queue = getenv('QUEUE') ?: 'validation-s23-php';
$baseNowMs = 1775400000000;
$gids = ['alpha', 'beta', 'gamma', 'delta', 'epsilon', 'zeta', 'eta'];

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    foreach ($gids as $idx => $gid) {
        $order = $idx + 1;
        $client->publish(
            queue: $queue,
            jobId: sprintf('%s-%s-job-001', $queue, $gid),
            payload: ['kind' => 'group-pagination', 'gid' => $gid, 'slot' => 1],
            gid: $gid,
            groupLimit: 1,
            nowMsOverride: $baseNowMs + $order,
        );
    }

    $page1 = $monitor->groupsReady($queue, offset: 0, limit: 3);
    $page2 = $monitor->groupsReady($queue, offset: 3, limit: 3);
    $scoredPage1 = array_map(static fn(GroupReady $item): array => groupReadyToArray($item), $monitor->groupsReadyWithScores($queue, offset: 0, limit: 3));
    $scoredPage2 = array_map(static fn(GroupReady $item): array => groupReadyToArray($item), $monitor->groupsReadyWithScores($queue, offset: 3, limit: 3));
    $status = array_map(static fn(GroupStatus $item): array => groupStatusToArray($item), $monitor->groupStatus($queue, ['alpha', 'delta', 'eta'], defaultLimit: 1));

    $groupsReadyRaw = array_map('strval', $redis->zRange(sprintf('{%s}:groups:ready', $queue), 0, -1) ?: []);

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'groups_ready_page_1' => $page1,
        'groups_ready_page_2' => $page2,
        'groups_ready_scored_page_1' => $scoredPage1,
        'groups_ready_scored_page_2' => $scoredPage2,
        'group_status' => $status,
        'groups_ready_raw' => $groupsReadyRaw,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
