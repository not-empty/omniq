<?php

declare(strict_types=1);

use Omniq\Client;
use Omniq\Helper;
use Omniq\RedisConnOpts;

require '/workspace/omniq-php/vendor/autoload.php';
require '/workspace/omniq/validation/_lib/php_redis.php';

$redisHost = getenv('REDIS_HOST') ?: 'omniq-redis';
$redisMode = getenv('REDIS_MODE') ?: 'standalone';

$key = getenv('KEY') ?: 'validation-s11-php';

$client = new Client(
    redisConnOpts: new RedisConnOpts(host: $redisHost, port: 6379),
    clientName: 'omniq-core-validation-php',
);
$redis = validation_raw_redis($redisHost, $redisMode);

try {
    $client->childsInit(key: $key, expected: 3);

    $ackSequence = [
        $client->childAck(key: $key, childId: 'a'),
        $client->childAck(key: $key, childId: 'a'),
        $client->childAck(key: $key, childId: 'b'),
        $client->childAck(key: $key, childId: 'c'),
    ];

    $base = substr(Helper::childsAnchor($key), 0, -5);
    $countExistsAfter = $redis->exists($base . ':count');
    $doneExistsAfter = $redis->exists($base . ':done');

    echo json_encode([
        'sdk' => 'php',
        'key' => $key,
        'ack_sequence' => $ackSequence,
        'count_exists_after' => (int) $countExistsAfter,
        'done_exists_after' => (int) $doneExistsAfter,
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
} finally {
    $client->close();
    $redis->close();
}
