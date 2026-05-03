<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\JobInfo;
use Omniq\LaneJob;
use Omniq\QueueMonitor;
use Omniq\QueueOverview;
use Omniq\QueueStats;
use Omniq\GroupReady;
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

function laneJobToArray(LaneJob $job): array
{
    return [
        'lane' => $job->lane,
        'job_id' => $job->jobId,
        'idx_score_ms' => $job->idxScoreMs,
        'state' => $job->state,
        'gid' => $job->gid,
        'attempt' => $job->attempt,
        'max_attempts' => $job->maxAttempts,
        'due_ms' => $job->dueMs,
        'lock_until_ms' => $job->lockUntilMs,
        'queued_ms' => $job->queuedMs,
        'first_started_ms' => $job->firstStartedMs,
        'last_started_ms' => $job->lastStartedMs,
        'completed_ms' => $job->completedMs,
        'failed_ms' => $job->failedMs,
        'updated_ms' => $job->updatedMs,
        'last_error' => $job->lastError,
    ];
}

function jobInfoToArray(JobInfo $job): array
{
    return [
        'job_id' => $job->jobId,
        'state' => $job->state,
        'gid' => $job->gid,
        'attempt' => $job->attempt,
        'max_attempts' => $job->maxAttempts,
        'timeout_ms' => $job->timeoutMs,
        'backoff_ms' => $job->backoffMs,
        'lease_token' => $job->leaseToken,
        'lock_until_ms' => $job->lockUntilMs,
        'due_ms' => $job->dueMs,
        'payload' => $job->payload,
        'last_error' => $job->lastError,
        'last_error_ms' => $job->lastErrorMs,
        'created_ms' => $job->createdMs,
        'updated_ms' => $job->updatedMs,
        'queued_ms' => $job->queuedMs,
        'first_started_ms' => $job->firstStartedMs,
        'last_started_ms' => $job->lastStartedMs,
        'completed_ms' => $job->completedMs,
        'failed_ms' => $job->failedMs,
    ];
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

function groupReadyToArray(GroupReady $item): array
{
    return [
        'gid' => $item->gid,
        'score_ms' => $item->scoreMs,
    ];
}

function overviewToArray(QueueOverview $overview): array
{
    return [
        'stats' => statsToArray($overview->stats),
        'ready_groups' => array_map(static fn(GroupReady $item): array => groupReadyToArray($item), $overview->readyGroups),
        'active' => array_map(static fn(LaneJob $job): array => laneJobToArray($job), $overview->active),
        'delayed' => array_map(static fn(LaneJob $job): array => laneJobToArray($job), $overview->delayed),
        'failed' => array_map(static fn(LaneJob $job): array => laneJobToArray($job), $overview->failed),
        'completed' => array_map(static fn(LaneJob $job): array => laneJobToArray($job), $overview->completed),
    ];
}

$queue = getenv('QUEUE') ?: 'validation-s14-php';
$baseNowMs = 1775310000000;

$waitKeep = $queue . '-wait-keep-001';
$waitMissing = $queue . '-wait-missing-001';
$activeJob = $queue . '-active-001';
$delayedJob = $queue . '-delayed-001';
$completedJob = $queue . '-completed-001';
$failedJob = $queue . '-failed-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$monitor = new QueueMonitor($client);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $completedJob, payload: ['kind' => 'monitor-lanes', 'slot' => 'completed'], nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $activeJob, payload: ['kind' => 'monitor-lanes', 'slot' => 'active'], nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $failedJob, payload: ['kind' => 'monitor-lanes', 'slot' => 'failed'], maxAttempts: 1, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $delayedJob, payload: ['kind' => 'monitor-lanes', 'slot' => 'delayed'], dueMs: $baseNowMs + 100000, nowMsOverride: $baseNowMs + 4);
    $client->publish(queue: $queue, jobId: $waitKeep, payload: ['kind' => 'monitor-lanes', 'slot' => 'wait-keep'], nowMsOverride: $baseNowMs + 5);
    $client->publish(queue: $queue, jobId: $waitMissing, payload: ['kind' => 'monitor-lanes', 'slot' => 'wait-missing'], nowMsOverride: $baseNowMs + 6);

    $completedRes = reserveJob($client, $queue, $baseNowMs + 100);
    reserveJob($client, $queue, $baseNowMs + 101);
    $failedRes = reserveJob($client, $queue, $baseNowMs + 102);

    $client->ackSuccess(queue: $queue, jobId: $completedRes->jobId, leaseToken: $completedRes->leaseToken, nowMsOverride: $baseNowMs + 150);
    $client->ackFail(queue: $queue, jobId: $failedRes->jobId, leaseToken: $failedRes->leaseToken, error: 'terminal failure', nowMsOverride: $baseNowMs + 151);

    $redis->del(sprintf('{%s}:job:%s', $queue, $waitMissing));

    $waitPage = array_map(static fn(LaneJob $job): array => laneJobToArray($job), $monitor->lanePage($queue, 'wait', offset: 0, limit: 10, reverse: false));
    $waitPageReverse = array_map(static fn(LaneJob $job): array => laneJobToArray($job), $monitor->lanePage($queue, 'wait', offset: 0, limit: 10, reverse: true));
    $findWait = array_map(static fn(LaneJob $job): array => laneJobToArray($job), $monitor->findJobs($queue, 'wait', [$waitKeep, $waitMissing]));
    $existingJob = $monitor->getJob($queue, $activeJob);
    $missingJob = $monitor->getJob($queue, $waitMissing);
    $overview = overviewToArray($monitor->overview($queue, samplesPerLane: 10));

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'wait_page' => $waitPage,
        'wait_page_reverse' => $waitPageReverse,
        'find_wait' => $findWait,
        'get_existing' => $existingJob instanceof JobInfo ? jobInfoToArray($existingJob) : null,
        'get_missing' => $missingJob,
        'overview' => $overview,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
