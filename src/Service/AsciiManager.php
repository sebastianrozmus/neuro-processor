<?php

namespace App\Service;

use RuntimeException;
use Symfony\Component\Filesystem\Filesystem;
use Symfony\Component\Process\Process;
use Symfony\Component\Process\Exception\ProcessFailedException;

class AsciiManager
{
    public function __construct()
    {
    }

    public function convertToAscii(string $filename): string
    {
        $filesystem = new Filesystem();

        if (!$filesystem->exists($filename)) {
            throw new RuntimeException(sprintf('File %s doesn\'t exist', $filename));
        }

        $command = [
            'ascii-image-converter',
            $filename,
            '-b', // TODO: parameters as a config
            '-C',
            '-c',
            '--bg-color'
        ];

        $process = new Process($command);
        $process->run();

        if (!$process->isSuccessful()) {
            throw new ProcessFailedException($process);
        }

        return $process->getOutput();
    }
}
