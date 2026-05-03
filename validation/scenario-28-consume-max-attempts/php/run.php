<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\JobCtx;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$queue = getenv('QUEUE') ?: 'validation-s28-php';
$jobId = $queue . '-job-001';
$baseNowMs = 1775440000000;

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$inspect = validation_raw_redis($redisHost, $redisMode);

$seen = [];
$signalPid = null;

try {
    $client->publish(
        queue: $queue,
        jobId: $jobId,
        payload: ['kind' => 'consume-max-attempts', 'sdk' => 'php'],
        maxAttempts: 3,
        backoffMs: 100,
        timeoutMs: 30000,
        nowMsOverride: $baseNowMs + 1,
    );

    $client->consume(
        queue: $queue,
        handler: static function (JobCtx $ctx) use (&$seen, &$signalPid): void {
            $isLastAttempt = $ctx->attempt >= $ctx->maxAttempts;
            $seen[] = [
                'attempt' => $ctx->attempt,
                'max_attempts' => $ctx->maxAttempts,
                'is_last_attempt' => $isLastAttempt,
            ];

            if (!$isLastAttempt) {
                throw new RuntimeException('Intentional failure before the last attempt');
            }

            if ($signalPid === null) {
                $pid = pcntl_fork();
                if ($pid === -1) {
                    throw new RuntimeException('failed to fork signal helper');
                }
                if ($pid === 0) {
                    usleep(50000);
                    posix_kill(posix_getppid(), SIGINT);
                    exit(0);
                }
                $signalPid = $pid;
            }
            usleep(100000);
        },
        pollIntervalS: 0.02,
        promoteIntervalS: 0.05,
        reapIntervalS: 10.0,
        drain: true,
    );

    echo json_encode([
        'sdk' => 'php',
        'queue' => $queue,
        'job_id' => $jobId,
        'seen' => $seen,
        'final_state' => (string) ($inspect->hGet(sprintf('{%s}:job:%s', $queue, $jobId), 'state') ?: ''),
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
