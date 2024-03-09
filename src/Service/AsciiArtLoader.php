<?php

namespace App\Service;

use RuntimeException;
use Symfony\Component\Process\Process;

class AsciiArtLoader
{
    public const ASCII_ART_FILE_EXTENSION = '.txt';

    public function __construct(private string $asciiArtPath)
    {
    }

    public function loadArt(string $filename): string
    {
        $path = join('', [
            $this->asciiArtPath,
            $filename,
            self::ASCII_ART_FILE_EXTENSION
        ]);

        $file = new SplFileInfo($filename, '', '');

        if (!file_exists($path)) {
            throw new RuntimeException(sprintf("File not found: %s", $path));
        }

        return $file->getContents();
    }
}
