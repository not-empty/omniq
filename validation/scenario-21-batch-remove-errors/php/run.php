<?php

declare(strict_types=1);

use Omniq\BatchResultItem;
use Omniq\Client;
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

/** @param list<BatchResultItem> $items */
function batchToArray(array $items): array
{
    return array_map(
        static fn(BatchResultItem $item): array => [
            'job_id' => $item->jobId,
            'status' => $item->status,
            'reason' => $item->reason,
        ],
        $items,
    );
}

$queue = getenv('QUEUE') ?: 'validation-s21-php';
$baseNowMs = 1775380000000;

$waitJob = $queue . '-wait-job-001';
$groupedWaitJob = $queue . '-grouped-wait-job-001';
$activeJob = $queue . '-active-job-001';
$delayedJob = $queue . '-delayed-job-001';
$missingJob = $queue . '-missing-job-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $activeJob, payload: ['kind' => 'batch-remove-errors', 'slot' => 'active'], maxAttempts: 3, nowMsOverride: $baseNowMs + 1);

    $activeRes = reserveJob($client, $queue, $baseNowMs + 100);
    if ($activeRes->jobId !== $activeJob) {
        throw new RuntimeException(sprintf('expected active job %s, got %s', $activeJob, $activeRes->jobId));
    }

    $client->publish(queue: $queue, jobId: $waitJob, payload: ['kind' => 'batch-remove-errors', 'slot' => 'wait'], maxAttempts: 3, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $groupedWaitJob, payload: ['kind' => 'batch-remove-errors', 'slot' => 'gwait'], maxAttempts: 3, gid: 'alpha', groupLimit: 1, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $delayedJob, payload: ['kind' => 'batch-remove-errors', 'slot' => 'delayed'], maxAttempts: 3, dueMs: $baseNowMs + 100000, nowMsOverride: $baseNowMs + 4);

    $batchRemoveResults = $client->removeJobsBatch(
        queue: $queue,
        lane: 'wait',
        jobIds: [$waitJob, $missingJob, $groupedWaitJob, $activeJob, $delayedJob],
    );

    $statsKey = sprintf('{%s}:stats', $queue);
    $stats = [
        'waiting' => (int) ($redis->hGet($statsKey, 'waiting') ?: 0),
        'group_waiting' => (int) ($redis->hGet($statsKey, 'group_waiting') ?: 0),
        'waiting_total' => (int) ($redis->hGet($statsKey, 'waiting_total') ?: 0),
        'active' => (int) ($redis->hGet($statsKey, 'active') ?: 0),
        'delayed' => (int) ($redis->hGet($statsKey, 'delayed') ?: 0),
        'groups_ready' => (int) ($redis->hGet($statsKey, 'groups_ready') ?: 0),
    ];

    $jobHashExists = [
        'wait_job' => (int) $redis->exists(sprintf('{%s}:job:%s', $queue, $waitJob)),
        'grouped_wait_job' => (int) $redis->exists(sprintf('{%s}:job:%s', $queue, $groupedWaitJob)),
        'active_job' => (int) $redis->exists(sprintf('{%s}:job:%s', $queue, $activeJob)),
        'delayed_job' => (int) $redis->exists(sprintf('{%s}:job:%s', $queue, $delayedJob)),
    ];

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'batch_remove_results' => batchToArray($batchRemoveResults),
        'job_hash_exists' => $jobHashExists,
        'stats' => $stats,
        'wait_len' => (int) $redis->lLen(sprintf('{%s}:wait', $queue)),
        'idx_wait' => (int) $redis->zCard(sprintf('{%s}:idx:wait', $queue)),
        'group_wait_len' => (int) $redis->lLen(sprintf('{%s}:g:alpha:wait', $queue)),
        'groups_ready' => (int) $redis->zCard(sprintf('{%s}:groups:ready', $queue)),
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
