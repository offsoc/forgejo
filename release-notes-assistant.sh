#!/bin/bash
# Copyright twenty-panda <twenty-panda@posteo.com>
# SPDX-License-Identifier: MIT

payload=$(mktemp)
pr=$(mktemp)
trap "rm $payload $pr" EXIT

cat >$payload
#
# If this is a backport, refer to the original PR to figure
# out the classification.
#
if $(jq --raw-output .IsBackportedFrom <$payload); then
  jq --raw-output '.BackportedFrom[0]' <$payload >$pr
else
  jq --raw-output '.Pr' <$payload >$pr
fi

labels=$(jq --raw-output '.labels[].name' <$pr)

#
# Was this PR labeled `worth a release note`?
#
if echo "$labels" | grep --quiet worth; then
  worth=true
else
  worth=false
fi

#
# If there was no release-notes/N.md file and it is not
# worth a release note, just forget about it.
#
if test -z "$(jq --raw-output .Draft <$payload)"; then
  if ! $worth; then
    echo -n ZA Included for completness but not worth a release note
    exit 0
  fi
fi

case "$labels" in
*bug*)
  if $(jq --raw-output .IsBackportedTo <$payload); then
    #
    # if it has been backported, it was in the release notes of an older stable release
    # and does not need to be in this more recent release notes
    #
    echo -n ZB Already announced in the release notes of an older stable release
    exit 0
  fi
  ;;
esac

case "$labels" in
*breaking*)
  case "$labels" in
  *feature*) echo -n AA Breaking features ;;
  *bug*) echo -n AB Breaking bug fixes ;;
  *) echo -n ZC Breaking changes without a feature or bug label ;;
  esac
  ;;
*forgejo/ui*)
  case "$labels" in
  *feature*) echo -n BA User Interface features ;;
  *bug*) echo -n BB User Interface bug fixes ;;
  *) echo -n ZD User Interface changes without a feature or bug label ;;
  esac
  ;;
*feature*) echo -n CA Features ;;
*bug*) echo -n CB Bug fixes ;;
*) echo -n ZE Other changes without a feature or bug label ;;
esac
