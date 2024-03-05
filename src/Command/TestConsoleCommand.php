<?php

namespace App\Command;

use App\Service\AsciiArtLoader;
use App\CLI\Console\ConsoleManager;
use InvalidArgumentException;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Terminal;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Console\Style\SymfonyStyle;
use Symfony\Component\Console\Cursor;


class TestConsoleCommand extends Command
{
    protected static string $name = 'console';

    public function __construct(private ConsoleManager $consoleManager)
    {
        parent::__construct();
    }

    protected function configure(): void
    {
        $this
            ->setDescription('Test console')
            ->setName(self::$name)
        ;
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $this->consoleManager->loop($input, $output);

        return Command::SUCCESS;
    }
}
