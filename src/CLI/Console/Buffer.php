<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;

// TODO: interfejs Buffer, klasy ImageBuffer, ScrollableBuffer, TextBuffer, WindowBuffer
class Buffer
{
    /**
     * @var array<array<int, string>|string>
     */
    private array $buffer;

    public function __construct(private int $width, private int $height, string $text)
    {
        $this->initBuffer();

        if ($text) {
            $this->write($text);
        }
    }

    public function initBuffer(): void
    {
        $this->buffer = [];

        for ($y = 0; $y <= $this->width; $y++) {
            for ($x = 0; $x <= $this->height; $x++) {
                $this->buffer[$x][$y] = ' ';
            }
        }

    }

    public function write(string $text, int $xOffset = 0, int $yOffset = 0, bool $isAsciiSequence = true): void
    {
        $lines = explode("\n", $text);
        foreach ($lines as $lineNumber => $line) {
            $characters = $isAsciiSequence ? $this->splitAnsiSequence($line) : str_split($line);

            for ($x = 0; $x < count($characters); $x++) {
                if (isset($this->buffer[$yOffset + $lineNumber][$xOffset + $x])) {
                    $this->buffer[$yOffset + $lineNumber][$xOffset + $x] = $characters[$x];
                }
            }
        }
    }

    // TODO: AsciiHelperService
    /**
     * @return array<int, string>
     */
    public function splitAnsiSequence(string $text): array
    {
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
            } elseif ($char === $escapeCharacter) {
                $actualCharacter .= $escapeCharacter;
            } elseif ($char === 'm') {
                $fetchedSequences++;
                $actualCharacter .= 'm';
            } else {
                $actualCharacter .= $char;
            }

            $i++;
        }

        $resultArray[] = $actualCharacter;

        return $resultArray;
    }
}
