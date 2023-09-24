# Hercules
## Overview
Hercules is a tool for hiring managers to vet if their technical assignment solutions are plagiarised from Github repositories. It is meant to help hiring managers sieve out solutions that do not help to achieve
What an assignment seeks to do: gauge your technical ability

## Installation

```
git clone https://github.com/ongteckwu/hercules.git
cd hercules
```

Next,
copy .env.example into .env
then add your own GITHUB_TOKEN= into .env

Build the application:
```
./build.sh
```

Run the application:
```
./hercules --url=https://github.com/xxx/yyyy

// OR

./hercules --dir=<path-to-code-directory>
```




## Contribution
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the MIT License - see the LICENSE.md file for details.