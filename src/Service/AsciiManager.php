<?php

namespace App\Service;

use RuntimeException;

class AsciiManager
{
    public function __construct(private AsciiArtLoader $asciiArtLoader)
    {
    }

    public function convertToAscii(string $filename): string
    {
        $filesystem = new Filesystem(); 

        if (!$filesystem->exists($filebane)) {
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
