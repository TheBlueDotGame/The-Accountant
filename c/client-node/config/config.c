///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <ctype.h>
#include <regex.h>
#include "config.h"
#include "../unit-test.h"

#define MAX_LINE_LENGTH 32768
#define MAX_TOKEN_NAME_LENGTH 8192
#define MAX_TOKEN_VALUE_LENGTH 24576

static char *trim_white_space(char *str)
{
    char *end;

    while(isspace((unsigned char)*str))
    {
        str++;
    }

    if(*str == 0)
    {
        return str;
    }
    
    end = str + strlen(str) - 1;
    while(end > str && isspace((unsigned char)*end))
    {
        end--;
    }

    end[1] = '\0';

    return str;
}

unit_static bool is_valid_url(char *url)
{
    regex_t re;
    // url_re matches http|https + :// + one or more non-white-space-characters + : + port number between 1 and 65535.
    if (regcomp(&re, "^https?:\\/\\/\\S+:([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$", REG_EXTENDED) != 0)
    {
        regfree(&re);
        return false;
    }
    if (regexec(&re, url, 0, NULL, 0) != 0)
    {
        regfree(&re);
        return false;
    }
        regfree(&re);
    return true;
}

static bool assign_to_config(Config *cfg, char *name, char *token)
{
    char *name_trimed = trim_white_space(name);
    char *token_trimed = trim_white_space(token);
    size_t len = strlen(token_trimed);
    
    if (len > MAX_TOKEN_VALUE_LENGTH)
    {
        return false;
    }

    if (strcmp("port", name_trimed) == 0)
    {
        int port = atoi(token_trimed);
        if (port < 1 || port > 65535)
        {
            return false;
        }
        cfg->port = port;
        return true;
    }
    
    if (strcmp("node_public_url", name_trimed) == 0)
    {
        if (!is_valid_url(token_trimed))
        {
            return false;
        }
        strncpy(cfg->node_public_url, token_trimed, len);
        return true;
    }

    if (strcmp("validator_url", name_trimed) == 0)
    {
        if (!is_valid_url(token_trimed))
        {
            return false;
        }
        strncpy(cfg->validator_url, token_trimed, len);
        return true;
    }

    if (strcmp("pem_file", name_trimed) == 0)
    {
        strncpy(cfg->pem_file, token_trimed, len);
        return true;
    }

    return false;
}

Config Config_new_from_file(char *file_path)
{
    FILE    *textfile;
    char    line[MAX_LINE_LENGTH];
    Config cfg;
     
    textfile = fopen(file_path, "r");
    if(textfile == NULL)
    {
        printf("Cannot open config file: %s\n", file_path);
        exit(1);
    }
     
    while(fgets(line, MAX_LINE_LENGTH, textfile)){
        char *line_trimed = trim_white_space(line);
        char *token = strtok(line_trimed, ":");
        size_t position = 0;
        char name[MAX_TOKEN_NAME_LENGTH];
        while(token != NULL && position < 2)
        {
            if (position == 0)
            {
                size_t len = strlen(token);
                if (len > MAX_TOKEN_NAME_LENGTH)
                {
                    printf("Max key name size exceeded, allowed %i, have %li\n", MAX_TOKEN_NAME_LENGTH, len);
                    exit(1);
                }
                strncpy(name, token, len);
                name[len] = '\0';
            } else {
                bool ok = assign_to_config(&cfg, name, token);
                if (!ok)
                {
                    printf("Given key name %s with value %s assign to config failed\n", name, token);
                    exit(1);
                }
            }
            token = strtok(NULL, "");
            position++;
        }
    }
     
    fclose(textfile);

    return  cfg;
}
