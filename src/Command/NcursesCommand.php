<?php

namespace App\Command;

use App\Enum\AsciiArtName;
use App\Service\AsciiArtLoader;
use InvalidArgumentException;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Console\Style\SymfonyStyle;

class NcursesCommand extends Command
{
    protected static string $name = 'ncurses';

    protected function configure(): void
    {
        $this
            ->setDescription('ncurses test')
            ->setName(self::$name)
        ;
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        ncurses_init();
        ncurses_border(0, 0, 0, 0, 0, 0, 0, 0);
        ncurses_refresh();
        sleep(5);
        ncurses_end();

        return Command::SUCCESS;
    }
}
