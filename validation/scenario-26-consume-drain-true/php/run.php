<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\JobCtx;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s26-php';
$baseNowMs = 1775430000000;
$firstJob = $queue . '-job-001';
$secondJob = $queue . '-job-002';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$inspect = validation_raw_redis($redisHost, $redisMode);

$handledJobIds = [];
$signalPid = null;

try {
    $client->publish(queue: $queue, jobId: $firstJob, payload: ['kind' => 'drain-true', 'slot' => 1], nowMsOverride: $baseNowMs + 1);
    $client->publish(queue: $queue, jobId: $secondJob, payload: ['kind' => 'drain-true', 'slot' => 2], nowMsOverride: $baseNowMs + 2);

    $client->consume(
        queue: $queue,
        handler: static function (JobCtx $ctx) use (&$handledJobIds, &$signalPid, $firstJob): void {
            $handledJobIds[] = $ctx->jobId;
            if ($ctx->jobId === $firstJob && $signalPid === null) {
                $pid = pcntl_fork();
                if ($pid === -1) {
                    throw new RuntimeException('failed to fork signal helper');
                }
                if ($pid === 0) {
                    usleep(100000);
                    posix_kill(posix_getppid(), SIGINT);
                    exit(0);
                }
                $signalPid = $pid;
            }
            usleep(750000);
        },
        pollIntervalS: 0.02,
        promoteIntervalS: 10.0,
        reapIntervalS: 10.0,
        drain: true,
    );

    $statsKey = sprintf('{%s}:stats', $queue);
    $stats = [
        'waiting' => (int) ($inspect->hGet($statsKey, 'waiting') ?: 0),
        'waiting_total' => (int) ($inspect->hGet($statsKey, 'waiting_total') ?: 0),
        'active' => (int) ($inspect->hGet($statsKey, 'active') ?: 0),
        'completed_kept' => (int) ($inspect->hGet($statsKey, 'completed_kept') ?: 0),
    ];

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'handled_job_ids' => $handledJobIds,
        'first_job_state' => (string) ($inspect->hGet(sprintf('{%s}:job:%s', $queue, $firstJob), 'state') ?: ''),
        'second_job_state' => (string) ($inspect->hGet(sprintf('{%s}:job:%s', $queue, $secondJob), 'state') ?: ''),
        'stats' => $stats,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    if (is_int($signalPid) && $signalPid > 0) {
        @posix_kill($signalPid, SIGTERM);
        @pcntl_waitpid($signalPid, $status, WNOHANG);
    }
    try {
        $client->close();
    } catch (Throwable) {
    }
    $inspect->close();
}
