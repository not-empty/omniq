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

$queue = getenv('QUEUE') ?: 'validation-s10-php';
$baseNowMs = 1775280000000;

$activeJob = $queue . '-active-job-001';
$singleRetryJob = $queue . '-single-retry-job-001';
$batchRetryJob = $queue . '-batch-retry-job-001';
$waitingRemoveJob = $queue . '-waiting-remove-job-001';
$delayedRemoveJob = $queue . '-delayed-remove-job-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $activeJob, payload: ['kind' => 'admin', 'slot' => 'active'], maxAttempts: 3, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $singleRetryJob, payload: ['kind' => 'admin', 'slot' => 'single-retry'], maxAttempts: 1, nowMsOverride: $baseNowMs + 2);
    $client->publish(queue: $queue, jobId: $batchRetryJob, payload: ['kind' => 'admin', 'slot' => 'batch-retry'], maxAttempts: 1, nowMsOverride: $baseNowMs + 3);
    $client->publish(queue: $queue, jobId: $waitingRemoveJob, payload: ['kind' => 'admin', 'slot' => 'waiting-remove'], maxAttempts: 3, nowMsOverride: $baseNowMs + 4);
    $client->publish(queue: $queue, jobId: $delayedRemoveJob, payload: ['kind' => 'admin', 'slot' => 'delayed-remove'], maxAttempts: 3, dueMs: $baseNowMs + 100000, nowMsOverride: $baseNowMs + 5);

    $activeRes = reserveJob($client, $queue, $baseNowMs + 100);
    $singleFailedRes = reserveJob($client, $queue, $baseNowMs + 101);
    $batchFailedRes = reserveJob($client, $queue, $baseNowMs + 102);

    $client->ackFail(queue: $queue, jobId: $singleFailedRes->jobId, leaseToken: $singleFailedRes->leaseToken, error: 'single retry setup', nowMsOverride: $baseNowMs + 150);
    $client->ackFail(queue: $queue, jobId: $batchFailedRes->jobId, leaseToken: $batchFailedRes->leaseToken, error: 'batch retry setup', nowMsOverride: $baseNowMs + 151);

    $client->retryFailed(queue: $queue, jobId: $singleRetryJob, nowMsOverride: $baseNowMs + 200);
    $batchRetryResults = $client->retryFailedBatch(queue: $queue, jobIds: [$batchRetryJob, $waitingRemoveJob], nowMsOverride: $baseNowMs + 201);

    try {
        $client->removeJob(queue: $queue, jobId: $activeJob, lane: 'wait');
        $removeActiveError = 'NO_ERROR';
    } catch (Throwable $exception) {
        $removeActiveError = $exception->getMessage();
    }

    $batchRemoveResults = $client->removeJobsBatch(queue: $queue, lane: 'wait', jobIds: [$waitingRemoveJob, $delayedRemoveJob]);
    $delayedRemoveResult = $client->removeJob(queue: $queue, jobId: $delayedRemoveJob, lane: 'delayed');

    $singleRetryKey = sprintf('{%s}:job:%s', $queue, $singleRetryJob);
    $singleRetryState = $redis->hGet($singleRetryKey, 'state');
    $singleRetryAttempt = $redis->hGet($singleRetryKey, 'attempt');

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'single_retry_state' => $singleRetryState === false ? '' : (string) $singleRetryState,
        'single_retry_attempt' => $singleRetryAttempt === false ? 0 : (int) $singleRetryAttempt,
        'batch_retry_results' => batchToArray($batchRetryResults),
        'remove_active_error' => $removeActiveError,
        'batch_remove_results' => batchToArray($batchRemoveResults),
        'delayed_remove_result' => $delayedRemoveResult,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
