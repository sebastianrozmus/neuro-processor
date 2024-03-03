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

class DisplayAsciiArtCommand extends Command
{
    protected static string $name = 'ascii';

    public function __construct(private AsciiArtLoader $asciiArtLoader)
    {
        parent::__construct();
    }

    protected function configure(): void
    {
        $this
            ->setDescription('Test show color ASCII Art')
            ->setName(self::$name)
            ->addArgument('filename', InputArgument::REQUIRED, 'File name')
        ;
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $filename = AsciiArtName::tryFrom($input->getArgument('filename'));

        if (null === $filename) {
            throw new InvalidArgumentException('Invalid ASCII Art file name');
        }

        $asciiArtContent = $this->asciiArtLoader->loadArt($filename->value);

        $io = new SymfonyStyle($input, $output);
        $io->writeln($asciiArtContent);

        $text = "Hello, this is a simulated human typing... Hello, this is a simulated human typing... \n
        Hello, this is a simulated human typing... Hello, this is a simulated human typing... Hello, this is a simulated human typing... 
        Hello, this is a simulated human typing... ";

        $terminal = new Terminal();
        $width = $terminal->getWidth();
        $height = $terminal->getHeight();

        $xOffset = intdiv($width, 2);
        $yOffset = intdiv($height, 10);

        $this->simulateTyping($text, $xOffset, $yOffset, $output);

        return Command::SUCCESS;
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
