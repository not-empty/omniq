<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\JobCtx;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

function childMain(): int
{
    global $redisHost, $redisMode;

    $queue = getenv('QUEUE');
    $markerStarted = getenv('MARKER_STARTED');
    $markerDone = getenv('MARKER_DONE');

    if ($queue === false || $markerStarted === false || $markerDone === false) {
        throw new RuntimeException('missing child env');
    }

    $client = new Client(
        redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
        clientName: 'omniq-core-validation-php-child',
    );
    $marker = validation_raw_redis($redisHost, $redisMode);

    try {
        $client->consume(
            queue: $queue,
            handler: static function (JobCtx $ctx) use ($marker, $markerStarted, $markerDone): void {
                $marker->set($markerStarted, '1');
                usleep(1500000);
                $marker->set($markerDone, '1');
            },
            pollIntervalS: 0.02,
            promoteIntervalS: 10.0,
            reapIntervalS: 10.0,
            drain: false,
        );

        return 0;
    } finally {
        try {
            $client->close();
        } catch (Throwable) {
        }
        $marker->close();
    }
}

function parentMain(): int
{
    global $redisHost, $redisMode;

    $queue = getenv('QUEUE') ?: 'validation-s27-php';
    $baseNowMs = 1775440000000;
    $firstJob = $queue . '-job-001';
    $secondJob = $queue . '-job-002';
    $markerStarted = sprintf('{%s}:marker:started', $queue);
    $markerDone = sprintf('{%s}:marker:done', $queue);

    $client = new Client(
        redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
        clientName: 'omniq-core-validation-php',
    );
    $inspect = validation_raw_redis($redisHost, $redisMode);

    try {
        $client->publish(queue: $queue, jobId: $firstJob, payload: ['kind' => 'drain-false', 'slot' => 1], nowMsOverride: $baseNowMs + 1);
        $client->publish(queue: $queue, jobId: $secondJob, payload: ['kind' => 'drain-false', 'slot' => 2], nowMsOverride: $baseNowMs + 2);

        $env = array_merge($_ENV, [
            'QUEUE' => $queue,
            'MARKER_STARTED' => $markerStarted,
            'MARKER_DONE' => $markerDone,
            'REDIS_HOST' => $redisHost,
            'REDIS_MODE' => $redisMode,
        ]);
        $cmd = ['php', '/workspace/omniq/validation/scenario-27-consume-drain-false/php/run.php', 'child'];

        $descriptors = [
            0 => ['pipe', 'r'],
            1 => ['pipe', 'w'],
            2 => ['pipe', 'w'],
        ];

        $proc = proc_open($cmd, $descriptors, $pipes, '/workspace/omniq-php', $env);
        if (!is_resource($proc)) {
            throw new RuntimeException('failed to start child process');
        }

        fclose($pipes[0]);
        stream_set_blocking($pipes[1], false);
        stream_set_blocking($pipes[2], false);

        try {
            $deadline = microtime(true) + 5.0;
            while (microtime(true) < $deadline) {
                if ($inspect->get($markerStarted) === '1') {
                    break;
                }
                usleep(50000);
            }

            $status = proc_get_status($proc);
            $childPid = $status['pid'] ?? 0;
            if ($childPid > 0) {
                posix_kill($childPid, SIGINT);
            }

            $exitCode = null;
            $deadline = microtime(true) + 5.0;
            while (microtime(true) < $deadline) {
                $status = proc_get_status($proc);
                if (!$status['running']) {
                    $exitCode = $status['exitcode'];
                    break;
                }
                usleep(50000);
            }

            if ($exitCode === null) {
                proc_terminate($proc, SIGKILL);
                $status = proc_get_status($proc);
                $exitCode = $status['exitcode'];
            }
        } finally {
            stream_get_contents($pipes[1]);
            stream_get_contents($pipes[2]);
            fclose($pipes[1]);
            fclose($pipes[2]);
            proc_close($proc);
        }

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
            'child_exit_code' => $exitCode,
            'handler_started' => $inspect->get($markerStarted) === '1',
            'handler_done' => $inspect->get($markerDone) === '1',
            'first_job_state' => (string) ($inspect->hGet(sprintf('{%s}:job:%s', $queue, $firstJob), 'state') ?: ''),
            'second_job_state' => (string) ($inspect->hGet(sprintf('{%s}:job:%s', $queue, $secondJob), 'state') ?: ''),
            'stats' => $stats,
        ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;

        return 0;
    } finally {
        try {
            $client->close();
        } catch (Throwable) {
        }
        $inspect->close();
    }
}

if (($argv[1] ?? '') === 'child') {
    exit(childMain());
}

exit(parentMain());
