<?php

namespace App\Service;

use RuntimeException;

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

        if (!file_exists($path)) {
            throw new RuntimeException(sprintf("File not found: %s", $path));
        }

        return file_get_contents($path);
    }
}
