<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;
use RuntimeException;

class ScrollabledBuffer extends Buffer
{
    private string $contents;

    private int $actualPosition = 0;

    private int $cursorLine = 0;

    private int $cursorPosition = 0;

    public function __construct(private int $width, private int $height, string $text, private bool $hasAutoScroll = true)
    {
        parent::initBuffer();

        if ($text) {
            $this->write($text);
        }
    }

    public function write($text, $xOffset = 0, $yOffset = 0, $isAsciiSequence = true)
    {
        if ($xOffset || $yOffset) {
            throw new RuntimeException('Scrollable buffer content can be only appended.');
        }

        $lines = explode("\n", $text);
        foreach ($lines as $lineNumber => $line) {
            $this->contents->addNewLine();
            $characters = $isAsciiSequence ? $this->splitAnsiSequence($line) : str_split($line);

            for ($x = 0; $x < count($characters); $x++) {
                if ($x >= $this->width) {
                    $this->contents->addNewLine();
                } else {
                    $this->contents .= $characters[$x];
                    ++$this->cursorPosition;
                }
            }
        }
    }

    public function updateBuffer(int $x, int $y)
    {
        $iterationCounter = 0;

        $startLine = $this->cursorLine > $height ? $this->cursorLine - $height : $height;

        for ($iterationCounter = 0; $iterationCounter < $height; $iterationCounter++) {
            $actualLine = $startLine + $iterationCounter;
            $buffer->write($this->contents[$actualLine], $x, $y + $iterationCounter);
        }
    }

    public function addNewLine()
    {
        ++$this->cursorLine;
        $this->contents .= \PHP_EOL;
    }
}
