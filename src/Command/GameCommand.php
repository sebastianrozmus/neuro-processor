<?php

namespace App\Command;

use App\Game\Game;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Console\Input\InputInterface;

class GameCommand extends Command
{
    protected static string $name = 'game:start';

    protected function configure()
    {
        $this
            ->setDescription('Starts the game.')
            ->setName(self::$name)
        ;
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        (new Game($output))->initialize();

        return Command::SUCCESS;
    }
}
