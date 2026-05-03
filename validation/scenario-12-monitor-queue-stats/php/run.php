<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\QueueMonitor;
use Omniq\QueueStats;
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

$prefix = getenv('PREFIX') ?: 'validation-s12-php';
$queueA = $prefix . '-paused';
$queueB = $prefix . '-mixed';
$baseNowMs = 1775290000000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);

try {
    $client->publish(queue: $queueA, jobId: $queueA . '-job-001', payload: ['kind' => 'monitor', 'queue' => 'a'], nowMsOverride: $baseNowMs + 1);
    $client->pause(queue: $queueA);

    $completedJob = $queueB . '-completed-job-001';
    $activeJob = $queueB . '-active-job-001';
    $delayedJob = $queueB . '-delayed-job-001';

    $client->publish(queue: $queueB, jobId: $completedJob, payload: ['kind' => 'monitor', 'slot' => 'completed'], nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queueB, jobId: $activeJob, payload: ['kind' => 'monitor', 'slot' => 'active'], nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queueB, jobId: $delayedJob, payload: ['kind' => 'monitor', 'slot' => 'delayed'], dueMs: $baseNowMs + 100000, nowMsOverride: $baseNowMs + 4);

    $completedRes = reserveJob($client, $queueB, $baseNowMs + 100);
    reserveJob($client, $queueB, $baseNowMs + 101);
    $client->ackSuccess(queue: $queueB, jobId: $completedRes->jobId, leaseToken: $completedRes->leaseToken, nowMsOverride: $baseNowMs + 150);

    $listQueues = $monitor->scanQueues();
    $queuesFound = array_values(array_filter(
        $listQueues,
        static fn(string $queue): bool => in_array($queue, [$queueA, $queueB], true),
    ));
    sort($queuesFound);
    $statsA = statsToArray($monitor->stats($queueA));
    $statsB = statsToArray($monitor->stats($queueB));
    $statsMany = array_map(
        static fn(QueueStats $stats): array => statsToArray($stats),
        $monitor->statsMany([$queueA, $queueB]),
    );

    echo json_encode([
        'sdk' => 'php',
        'queues_found' => $queuesFound,
        'stats_a' => $statsA,
        'stats_b' => $statsB,
        'stats_many' => $statsMany,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
}
