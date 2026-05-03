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

$queue = getenv('QUEUE') ?: 'validation-s20-php';
$baseNowMs = 1775370000000;

$failedJob = $queue . '-failed-job-001';
$waitingJob = $queue . '-waiting-job-001';
$missingJob = $queue . '-missing-job-001';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->publish(queue: $queue, jobId: $failedJob, payload: ['kind' => 'batch-retry-errors', 'slot' => 'failed'], maxAttempts: 1, nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $waitingJob, payload: ['kind' => 'batch-retry-errors', 'slot' => 'waiting'], maxAttempts: 3, nowMsOverride: $baseNowMs + 2);

    $failedRes = reserveJob($client, $queue, $baseNowMs + 100);
    $client->ackFail(queue: $queue, jobId: $failedRes->jobId, leaseToken: $failedRes->leaseToken, error: 'make failed', nowMsOverride: $baseNowMs + 150);

    $batchRetryResults = $client->retryFailedBatch(queue: $queue, jobIds: [$failedJob, $missingJob, $waitingJob], nowMsOverride: $baseNowMs + 200);

    $retriedJobState = $redis->hGet(sprintf('{%s}:job:%s', $queue, $failedJob), 'state');

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'batch_retry_results' => batchToArray($batchRetryResults),
        'retried_job_state' => $retriedJobState === false ? '' : (string) $retriedJobState,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
