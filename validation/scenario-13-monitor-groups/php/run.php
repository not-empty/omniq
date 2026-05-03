<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\GroupReady;
use Omniq\GroupStatus;
use Omniq\QueueMonitor;
use Omniq\RedisConnOpts;
use Omniq\ReserveJob;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

function reserveJob(Client $client, string $queue, int $nowMs): ReserveJob
{
    $result = $client->reserve(queue: $queue, nowMsOverride: $nowMs);
    if (!$result instanceof ReserveJob || $result->status !== 'JOB') {
        throw new RuntimeException('unexpected reserve response');
    }

    return $result;
}

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

$queue = getenv('QUEUE') ?: 'validation-s13-php';
$baseNowMs = 1775300000000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);

try {
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-001', payload: ['kind' => 'monitor-groups', 'slot' => 'alpha-1'], gid: 'alpha', groupLimit: 2, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $queue . '-alpha-job-002', payload: ['kind' => 'monitor-groups', 'slot' => 'alpha-2'], gid: 'alpha', groupLimit: 2, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $queue . '-beta-job-001', payload: ['kind' => 'monitor-groups', 'slot' => 'beta-1'], gid: 'beta', groupLimit: 1, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $queue . '-gamma-job-001', payload: ['kind' => 'monitor-groups', 'slot' => 'gamma-1'], gid: 'gamma', groupLimit: 1, nowMsOverride: $baseNowMs + 4);
    $client->publish(queue: $queue, jobId: $queue . '-delta-job-001', payload: ['kind' => 'monitor-groups', 'slot' => 'delta-1'], gid: 'delta', groupLimit: 1, nowMsOverride: $baseNowMs + 5);

    reserveJob($client, $queue, $baseNowMs + 100);

    $gids = ['alpha', 'beta', 'gamma', 'delta'];
    $groupsReadyPage = $monitor->groupsReady($queue, offset: 0, limit: 2);
    $groupsReadyAll = array_map(
        static fn(GroupReady $item): array => groupReadyToArray($item),
        $monitor->groupsReadyWithScores($queue, offset: 0, limit: 10),
    );
    $groupStatus = array_map(
        static fn(GroupStatus $item): array => groupStatusToArray($item),
        $monitor->groupStatus($queue, $gids, defaultLimit: 1),
    );

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'groups_ready_page' => $groupsReadyPage,
        'groups_ready_all' => $groupsReadyAll,
        'group_status' => $groupStatus,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
