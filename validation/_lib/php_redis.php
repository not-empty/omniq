<?php

declare(strict_types=1);

function validation_raw_redis(string $host, string $mode): Redis|RedisCluster
{
    if ($mode === 'cluster') {
        return new RedisCluster(
            null,
            [$host . ':6379'],
            0.0,
            0.0,
            false,
            null,
            null,
        );
    }

    $redis = new Redis();
    $redis->connect($host, 6379);

    return $redis;
}

function validation_script_flush(Redis|RedisCluster $redis): void
{
    if ($redis instanceof RedisCluster) {
        foreach ($redis->_masters() as $master) {
            $host = (string) $master[0];
            $port = (int) $master[1];

            $node = new Redis();
            $node->connect($host, $port);
            $node->rawCommand('SCRIPT', 'FLUSH');
            $node->close();
        }

        return;
    }

    $redis->rawCommand('SCRIPT', 'FLUSH');
}
