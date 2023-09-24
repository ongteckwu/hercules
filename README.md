# Hercules
<p align="center">
  <img src="https://github.com/ongteckwu/hercules/assets/3834724/d12eca5b-ffff-4875-af66-e277be3a3124" />
</p>

**Hercules is a tool for hiring managers.**

It checks if job applicants have copied their tech assignments from GitHub.

The GOAL? 

To make sure assignments really show how skilled someone is in tech, not just how good their search, copy, and paste skills are 🚫📑.

## Backstory

Someone submitted an assignment that looked too well-done for the amount of time given. 

My suspicion made me CSI into Github, and that led me to finding the repositories he cloned. 

Not wanting to have to do this manual work again, I decided to just build a tool to automate the process 🤓.

## Sample Output
![download](https://github.com/ongteckwu/hercules/assets/3834724/b464b537-8c85-4f91-9b6c-ebfe3b020d37)

## Installation

```
git clone https://github.com/ongteckwu/hercules.git
cd hercules

//copy .env.example into .env
// then add your own GITHUB_TOKEN= into .env
```

Build the application:
```
./build.sh
```

Run the application:
```
./hercules --url=https://github.com/xxx/yyyy
```
OR
```
./hercules --dir=<path-to-code-directory>
```

Note: The application will take a couple of mins to run, due to the sheer volume of code to scan, and also to Github API limits.

## How it works

### Initialize

1️⃣ Randomly picks N=15 code files from the assignment (ignores files that aren't code and folders).

### Keywords Search
2️⃣ Next, it uses token-level TFIDF to identify key terms in the code. 

These terms are used to search GitHub for similar code files. 

The top 10 best matched Github files are pulled per assignment code file. 🕵️‍♂️

### Similarity Metrics

3️⃣ Next, it applies two different methods, Double-side Argmin Levenshtein (DAL) and Char-level non-alphabet TFIDF (CLNAT), to see how similar the assignment code file is to the GitHub code files. 🔍 

#### Double-side Argmin Levenshtein (DAL)
A form of Levenshtein that is used to find the most similar substring of string 1 in another string 2 (argmin). 
It returns:
1. the substring correction count (the min) and
2. the index of where the substring ends (the argmin).

Using it on the reverse of string 1 and string 2 gives the index of where the substring starts (double-sided).

The similarity score is `score = 1 - min/(sub_string_char_count)`

#### Char-level non-alphabet TFIDF
This is TFIDF but on a character level. It ignores alphabets so that it is variable-name-change invariant.

### Repo-to-repo match

4️⃣ It then counts the number of Github repositories that have similar code files. 

Then, it picks the top M=8 similar repositories and compares them directly to the assignment using both DAL and CLNAT. 

This results in three scores: summed weighted DAL, summed weighted CLNAT, and a weighted Combined Similarity. 📊

#### Weighted
Weighted by character count e.g. `sum(character_count[i]/total_character_count * similarity_score[i])`

#### Combined Similarity Score
`combined_similarity_score = dal_score * clnat_score`

### Show Results
5️⃣  Finally, it ranks these GitHub repositories based on the combined similarity score and shows the results in a table. 📝

Note: N and M can be tuned.

## Contribution
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the MIT License - see the LICENSE.md file for details.
