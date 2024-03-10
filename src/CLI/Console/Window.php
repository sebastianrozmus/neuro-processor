<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;

class Window
{
    public function __construct(
        private int $x,
        private int $y,
        private int $width,
        private int $height,
        private BorderStyle $borderStyle,
        private int $backgdoundColor = 0,
        ?Buffer $buffer = null
    ) {
        $this->buffer = $buffer ? $buffer : new Buffer($width, $height);
    }

    public function getBuffer()
    {
        return $buffer;
    }

    public function draw()
    {
        // TODO: draw border
        // TODO: draw background
        // TODO: draw buffer
    }

}
