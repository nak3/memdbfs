#!/bin/bash

cd `dirname $0`
source common.sh

mkdir ${testdir}/dir1
mkdir ${testdir}/dir1/dir2
touch ${testdir}/dir1/file1
echo "foo bar" > ${testdir}/dir1/file2
cat ${testdir}/dir1/file2
rm ${testdir}/dir1/file2
tree ${testdir}

echo "test success!"
echo "
${testdir}
└── dir1
    ├── dir2
    └── file1
"
