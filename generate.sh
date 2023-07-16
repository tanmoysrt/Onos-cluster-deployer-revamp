#!/bin/sh


userid=$(id -u)
if ! [ $userid -eq 0 ]; then
  echo "You must run this as root user"
  exit 1
fi

current_directory=$(pwd)/current
store_directory=$(pwd)/generated

rm -r $store_directory 2> /dev/null
mkdir $store_directory

# Compile first
subdirectories=$(find "$current_directory" -maxdepth 1 -type d -not -path "$current_directory" -not -name "*m2" -not -name "*generated")
for subdirectory in $subdirectories; do
   rm -r "$subdirectory/target" 2> /dev/null
   mkdir "$subdirectory/target"
   docker run -it --rm --name test-maven -v "$(pwd)/m2":/root/.m2  -v "$subdirectory":/usr/src/mymaven -w /usr/src/mymaven maven:3.3-jdk-8 mvn clean install -q -B
done

# Move oar files
for subdirectory in $subdirectories; do
  oar_files=$(find "$subdirectory/target" -type f -name "*.oar")
  for file in $oar_files; do
    mv $file $store_directory
    break
  done
done

# Delete target folder
for subdirectory in $subdirectories; do
   rm -r "$subdirectory/target" 2> /dev/null
done

# Delete m2 folder
rm -r "$(pwd)/m2"

# Make generated files accessible to normal user
files=$(find $store_directory -type f -name "*.oar")
for file in $files; do
  echo "> $file"
  chmod 777 $file
done

# Done !
echo "All work done !"
