<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;

class Buffer
{
    private array $buffer;

    public function __construct(private int $width, private int $height, string $text) {
        $this->initBuffer();

        if ($text) {
            $this->write($text);
        }
    }

    public function initBuffer()
    {
        $this->buffer = [];

        for ($y=0; $y <= $this->width; $y++) { 
            for ($x=0; $x <= $this->height; $x++) { 
                $this->buffer[$x][$y] = ' ';
            }
        }

    }

    public function write($text, $xOffset = 0, $yOffset = 0)
    {
        $lines = explode("\n", $text);
        foreach ($lines as $lineNumber => $line) {
            $characters = explode('m', $line);
            $characters = explode("\033", $line);
            $characters = $this->splitAnsiSequence($line);

            for ($x = 0; $x < count($characters); $x++) {
                if (isset($this->buffer[$yOffset + $lineNumber][$xOffset + $x])) {
                    $this->buffer[$yOffset + $lineNumber][$xOffset + $x] = $characters[$x];
                }
            }
        }
    }

    public function splitAnsiSequence($text) {
        $length           = mb_strlen($text, 'UTF-8');
        $resultArray      = [];
        $i                = 0;
        $escapeCharacter  = "\x1B";
        $actualCharacter  = '';
        $fetchedSequences = 0;

        while ($i < $length) {
            $char = mb_substr($text, $i, 1, 'UTF-8');

            if ($char === $escapeCharacter && $fetchedSequences === 2) {
                $resultArray[]    = $actualCharacter;
                $fetchedSequences = 0;
                $actualCharacter  = $escapeCharacter;
            }
            elseif ($char === $escapeCharacter) {
                $actualCharacter .= $escapeCharacter;
            }
            elseif ($char === 'm') {
                $fetchedSequences++;
                $actualCharacter .= 'm';
            }
            else {
                $actualCharacter .= $char;
            }

            $i++;
        }

        $resultArray[] = $actualCharacter;

        return $resultArray;
    }
}
