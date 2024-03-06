<?php

namespace App\CLI\Console;

use App\Service\AsciiArtLoader;
use Symfony\Component\Console\Cursor;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Process\Process;
use Symfony\Component\Console\Style\SymfonyStyle;

class ConsoleManager
{
    private int $screenWidth = 0;
    private int $screenHeight = 0;
    private array $buffer;

    public function __construct(
        private Terminal $terminal,
        private AsciiArtLoader $asciiArtLoader,
    ) {
        $this->initScreen();
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
        $this->buffer       = [];

        for ($x=0; $x < $this->screenWidth; $x++) { 
            for ($y=0; $y < $this->screenHeight; $y++) { 
                $this->buffer[$x][$y] = ' ';
            }
        }

        return $hasChanged;
    }

    public function getChar(): string
    {
        // disable blocking: https://www.hashbangcode.com/article/creating-game-php-part-1-detecting-key-inputs
        // TODO: stream_set_blocking($stdin, 0);

        // switch terminal mode to raw and turn off echo
        system('stty cbreak -echo');

        // read character
        $key = fgetc(STDIN);

        // revert terminal mode
        system('stty cooked echo');

        return $key;
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

    public function loop($input, $output)
    {
        $cursor = new Cursor($output);
        // TODO: asset manager
        $forest = $this->asciiArtLoader->loadArt('forest1');
        $npc    = $this->asciiArtLoader->loadArt('npc1');

        $io = new SymfonyStyle($input, $output);
        $lastChar = "Moria to tajemnicza handlarka artefaktami, której towar często pochodzi z\nniebezpiecznych wykopalisk. Jest bardzo przebiegła i doskonale\nwie, jak wykorzystać wiedzę o przedmiotach, by maksymalizować zysk. Targowanie się z\nnią wymaga ostrożności i znajomości historii przedmiotów."."\n\n";
        do {
            $output->write(sprintf("\033\143"));

            $io->writeln($forest);

            //$this->terminal->clear();
            $this->initScreen();

            $cursor->moveToPosition($this->screenWidth / 2 + 20, 25);
            $lines = explode("\n", $lastChar);
            if (count($lines) + 25 >= $this->screenHeight) {
                //array_shift($lines);
                unset($lines[0], $lines[1], $lines[2], $lines[3], $lines[4]);
                $lastChar = join("\n", $lines);
            }
            foreach ($lines as $k => $line) {
                $cursor->moveToPosition($this->screenWidth / 2 + 20, 25 + $k);
                $output->write($line);
            }

            $cursor->moveToPosition($this->screenWidth / 2 + 60, 50);
            $this->offsetWrite($output, $cursor, $npc, $this->screenWidth / 2 + 40, 2);

            // TODO: $this->drawWidgets();
            $char = $this->getChar();

            if (ord($char) === 10) {
                $lastChar .= "\n\nLustrzany Medalion Czasu\n\nPozwala użytkownikowi na krótkie podróże w czasie, nie dłuższe niż kilka minut wstecz.\nUżytkowanie jest ograniczone i wymaga dużej ilości energii magicznej.";
                //$lastChar .= str_repeat(' ', $this->screenWidth / 2 + 60);
            }

            $lastChar .= $char;
            //$output->write('|asdasdasd');
            $this->onKeyPressed($char);
            //$this->widgets->emit(ConsoleEvents::KEY_PRESSED, new KeyPressedEvent($char));
        } while (!$this->shoudExit($char));
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

    public function offsetWrite($output, $cursor, string $text, int $offsetX = 0, $offsetY = 0): void
    {
        $lines = explode("\n", $text);

        foreach ($lines as $k => $line) {
            $cursor->moveToPosition($offsetX, $offsetY + $k);
            $output->write($line);
        }
    }

    private function simulateTyping(string $text, int $xOffset, int $yOffset, OutputInterface $output): void
    {
        for ($i = 0; $i < $yOffset; $i++) {
            $output->writeln('');
        }

        $spaces = str_repeat(' ', $xOffset);
        foreach (str_split($text) as $char) {
            $output->write($spaces . $char);
            $spaces = '';
            usleep(10000);
        }

        $output->writeln('');
    }
}
