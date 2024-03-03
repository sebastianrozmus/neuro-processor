FROM php:7.4

RUN apt-get update && \
    apt-get install -y \
        git \
        curl \
        vim \
        libncurses5-dev \
        ncurses-doc \
        libncursesw5-dev \
        zlib1g-dev \
        wget \
        libzip-dev \
    && apt-get clean

RUN docker-php-ext-install zip

RUN php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');" \
    && php -r "if (hash_file('SHA384', 'composer-setup.php') === 'dac665fdc30fdd8ec78b38b9800061b4150413ff2e3b6f88543c636f7cd84f6db9189d43a81e5503cda447da73c7e5b6') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;" \
    && php composer-setup.php \
    && mv composer.phar /usr/local/bin/composer \
    && php -r "unlink('composer-setup.php');"

RUN cd ~/ && \
    pecl download ncurses && \
    mkdir ~/ncurses && \
    cd ~/ncurses && \
    tar -xvzf ~/ncurses-1.0.2.tgz && \
    wget "https://bugs.php.net/patch-display.php?bug_id=71299&patch=ncurses-php7-support-again.patch&revision=1474549490&download=1" -O ~/ncurses/ncurses.patch && \
    mv ncurses-1.0.2 ncurses-php5 && \
    patch --strip=0 --verbose --ignore-whitespace < ~/ncurses/ncurses.patch && \
    cd ncurses-php5 && \
    phpize && \
    ./configure && \
    make && \
    make install && \
    docker-php-ext-enable ncurses && \
    rm -Rf ~/ncurses ~/ncurses-1.0.2 ~/channels.xml

WORKDIR /app
