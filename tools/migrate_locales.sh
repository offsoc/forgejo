#!/bin/bash

# Copyright 2024 The Forgejo Authors. All rights reserved.
# SPDX-License-Identifier: MIT

if [ -z "$1" ] || [ -z "$2" ]
then
	echo "USAGE: $0 section key [key1 [keyN]]"
	exit 1
fi

if ! [ -d ../options/locale_next ]
then
	echo 'Call this script from the `tools` directory.'
	exit 1
fi

destsection="$1"
keyJSON="$destsection.$2"
key1=""
keyN=""
if [ -n "$3" ]
then
	key1="$3"
else
	key1="$2"
fi
if [ -n "$4" ]
then
	keyN="$4"
fi

cd ../options/locale

# Migrate the string in one file.
function process() {
	file="$1"
	exec 3<$file

	val1=""
	valN=""
	cursection=""
	line1=0
	lineN=0
	lineNumber=0

	# Parse the file
	while read -u 3 line
	do
		((++lineNumber))
		if [[ $line =~ ^\[[-._a-zA-Z0-9]+\]$ ]]
		then
			cursection="${line#[}"
			cursection="${cursection%]}"
		elif [ "$cursection" = "$destsection" ]
		then
			key="${line%%=*}"
			value="${line#*=}"
			key="$(echo $key)"  # Trim leading/trailing whitespace
			value="$(echo $value)"

			if [ "$key" = "$key1" ]
			then
				val1="$value"
				line1=$lineNumber
			fi
			if [ -n "$keyN" ] && [ "$key" = "$keyN" ]
			then
				valN="$value"
				lineN=$lineNumber
			fi

			if [ -n "$val1" ] && ( [ -n "$valN" ] || [ -z "$keyN" ] )
			then
				# Found all desired strings
				break
			fi
		fi
	done

	if [ -n "$val1" ] || [ -n "$valN" ]
	then
		localename="${file#locale_}"
		localename="${localename%.ini}"
		localename="${localename%-*}"

		if [ "$file" = "locale_en-US.ini" ]
		then
			# Delete migrated string from source file
			if [ $line1 -gt 0 ] && [ $lineN -gt 0 ] && [ $lineN -ne $line1 ]
			then
				sed -i "${line1}d;${lineN}d" "$file"
			elif [ $line1 -gt 0 ]
			then
				sed -i "${line1}d" "$file"
			elif [ $lineN -gt 0 ]
			then
				sed -i "${lineN}d" "$file"
			fi
		fi

		# Write JSON
		jsonfile="../locale_next/${file/.ini/.json}"

		pluralform="other"
		oneform="one"
		case $localename in
			"be" | "bs" | "cnr" | "csb" | "hr" | "lt" | "pl" | "ru" | "sr" | "szl" | "uk" | "wen")
				# These languages have no "other" form and use "many" instead.
				pluralform="many"
				;;
			"ace" | "ay" | "bm" | "bo" | "cdo" | "cpx" | "crh" | "dz" | "gan" | "hak" | "hnj" | "hsn" | "id" | "ig" | "ii" | "ja" | "jbo" | "jv" | "kde" | "kea" | "km" | "ko" | "kos" | "lkt" | "lo" | "lzh" | "ms" | "my" | "nan" | "nqo" | "osa" | "sah" | "ses" | "sg" | "son" | "su" | "th" | "tlh" | "to" | "tok" | "tpi" | "tt" | "vi" | "wo" | "wuu" | "yo" | "yue" | "zh")
				# These languages have no singular form.
				oneform=""
				;;
			*)
				;;
		esac

		content=""
		if [ -z "$keyN" ]
		then
			content="$(jq --arg val "$val1" ".$keyJSON = \$val" < "$jsonfile")"
		else
			object='{}'
			if [ -n "$val1" ] && [ -n "$oneform" ]
			then
				object=$(jq --arg val "$val1" ".$oneform = \$val" <<< "$object")
			fi
			if [ -n "$valN" ]
			then
				object=$(jq --arg val "$valN" ".$pluralform = \$val" <<< "$object")
			fi
			content="$(jq --argjson val "$object" ".$keyJSON = \$val" < "$jsonfile")"
		fi
		jq . <<< "$content" > "$jsonfile"
	fi
}

for file in *.ini
do
	process "$file" &
done
wait

