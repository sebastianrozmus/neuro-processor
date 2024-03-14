<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;

class ConsoleManager
{
    private int $screenWidth = 0;
    private int $screenHeight = 0;
    private array $buffer;

    private Cursor $cursor;
    private OutputInterface $output;
    private InputInterface $input;

    public function __construct(
        private Terminal $terminal,
        private AsciiArtLoader $asciiArtLoader,
    ) {
        $this->initScreen();
    }

    public function setInput(InputInterface $input)
    {
        $this->input = $input;
    }

    public function setOutput(OutputInterface $output)
    {
        $this->output = $output;
    }

    public function setCursor(Cursor $cursor)
    {
        $this->cursor = $cursor;
    }

    public function initScreen(): bool
    {
        // TODO: it looks that it is never changing
        $hasChanged = $this->screenWidth !== $this->terminal->getWidth() ||
                      $this->screenHeight !== $this->terminal->getHeight();

        if ($hasChanged) {
            // TODO: $this->widgets->emit(ConsoleEvents::SCREEN_CHANGED, ScreenChangedEvent($this->screenWidth, $this->screenHeight, $this->terminal->getWidth(), $this->terminal->getHeight()))
        }

        $this->screenWidth  = $this->terminal->getWidth();
        $this->screenHeight = $this->terminal->getHeight();
        $this->initBuffer();

        return $hasChanged;
    }

    public function initBuffer()
    {
        $this->buffer = [];

        for ($y = 0; $y <= $this->screenWidth; $y++) {
            for ($x = 0; $x <= $this->screenHeight; $x++) {
                $this->buffer[$x][$y] = ' ';
            }
        }
    }

    public function write($text, $xOffset = null, $yOffset = null)
    {
        if (null === $xOffset || null === $yOffset) {
            $position = $this->cursor->getCurrentPosition();
            $xOffset  = $xOffset ? $xOffset : $position[0];
            $yOffset  = $yOffset ? $yOffset : $position[1];
        }

        $lines = explode("\n", $text);
        foreach ($lines as $lineNumber => $line) {
            $characters = explode('m', $line);

            $characters = explode("\033", $line);

            for ($x = 0; $x < count($characters); $x++) {
                if (isset($this->buffer[$yOffset + $lineNumber][$xOffset + $x])) {
                    $this->buffer[$yOffset + $lineNumber][$xOffset + $x] = $characters[$x];
                }
            }
        }

    }

    public function refreshScreen()
    {
        $position = $this->cursor->getCurrentPosition();

        foreach ($this->buffer as $lineNumber => $line) {
            $this->cursor->moveToPosition(0, $lineNumber);
            $this->output->write($line);
        }

        $this->cursor->moveToPosition($position[0], $position[1]);
    }

    public function nonBlockingRead($fd, &$data)
    {
        $read = array($fd);
        $write = array();
        $except = array();

        $result = stream_select($read, $write, $except, 0);
        if($result === false) {
            throw new RuntimeException('stream_select failed');
        }

        if(0 === $result) {
            return false;
        }

        $data = stream_get_line($fd, 1);

        return true;
    }

    public function setRawTerminalMode(bool $state): void
    {
        $process = new Process([
            'stty',
            $state ? 'cbreak' : 'cooked',
            $state ? '-echo' : 'echo',
        ]);
        $process->run();
    }

    public function loop()
    {
        // TODO: asset manager
        $forest = $this->asciiArtLoader->loadArt('forest1');
        $npc    = $this->asciiArtLoader->loadArt('npc1');
        $this->cursor->clearScreen();
        // $this->write($npc, 145, 4);
        // $this->write($forest, 0, 0);

        // $this->refreshScreen();
        // die();

        $io = new SymfonyStyle($this->input, $this->output);
        $lastChar = "Moria to tajemnicza handlarka artefaktami, której towar często pochodzi z\nniebezpiecznych wykopalisk. Jest bardzo przebiegła i doskonale\nwie, jak wykorzystać wiedzę o przedmiotach, by maksymalizować zysk. Targowanie się z\nnią wymaga ostrożności i znajomości historii przedmiotów."."\n\n";

        $stdin = fopen('php://stdin', 'r');
        stream_set_blocking($stdin, false);

        // switch terminal mode to raw and turn off echo
        system('stty cbreak -echo');

        do {
            $this->cursor->clearScreen();
            $this->write($forest, 1, 1);

            //$this->terminal->clear();
            $this->initScreen();

            $this->cursor->moveToPosition($this->screenWidth / 2 + 20, 25);
            $lines = explode("\n", $lastChar);
            if (count($lines) + 25 >= $this->screenHeight) {
                //array_shift($lines);
                unset($lines[0], $lines[1], $lines[2], $lines[3], $lines[4]);
                $lastChar = join("\n", $lines);
            }
            foreach ($lines as $k => $line) {
                $this->cursor->moveToPosition($this->screenWidth / 2 + 20, 25 + $k);
                $this->write($line);
            }

            $this->cursor->moveToPosition($this->screenWidth / 2 + 60, 50);
            $this->offsetWrite($npc, $this->screenWidth / 2 + 40, 2);

            // TODO: $this->drawWidgets();
            $char = false;
            $this->nonBlockingRead($stdin, $char);

            if (ord($char) === 10) {
                $lastChar .= "\n\nLustrzany Medalion Czasu\n\nPozwala użytkownikowi na krótkie podróże w czasie, nie dłuższe niż kilka minut wstecz.\nUżytkowanie jest ograniczone i wymaga dużej ilości energii magicznej.";
                //$lastChar .= str_repeat(' ', $this->screenWidth / 2 + 60);
            }

            $lastChar .= $char;
            //$output->write('|asdasdasd');
            $this->onKeyPressed($char);
            $this->refreshScreen();
            //$this->widgets->emit(ConsoleEvents::KEY_PRESSED, new KeyPressedEvent($char));
        } while (!$this->shoudExit($char));

        // revert terminal mode
        system('stty cooked echo');
    }

    public function onKeyPressed(string $key)
    {
        echo 'DEBUG: '.$key.' '.ord($key)."\n";
        // foreach ($this->getWidgets() as $widget) {
        // 	$widget->onKeuPressed($key);
        // }
    }

    public function shoudExit($key): bool
    {
        return ord($key) === 27;
    }

    public function offsetWrite(string $text, int $offsetX = 0, $offsetY = 0): void
    {
        $lines = explode("\n", $text);

        foreach ($lines as $k => $line) {
            $this->write($line, $offsetX, $offsetY + $k);
        }
    }

    private function simulateTyping(string $text, int $xOffset, int $yOffset, OutputInterface $output): void
    {
        $x = 0;
        foreach (str_split($text) as $char) {
            $this->write($char, $xOffset, $yOffset + $x++);
            $spaces = '';
            usleep(10000);
        }
    }
}
