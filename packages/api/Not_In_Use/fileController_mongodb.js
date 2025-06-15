const { gridfsBucket } = require("../Model/Schema");
const helper = require("../Utilities/helper");
const jwt = require("jsonwebtoken");
const fs = require('fs');
const path = require('path');

exports.uploadFile() = async (req, res) => {

    fs.createReadStream('./path/to/your/file.txt')
    .pipe(bucket.openUploadStream('file.txt', {
      chunkSizeBytes: 1048576, // Optional: specify chunk size
      metadata: { field: 'myField', value: 'myValue' } // Optional: add metadata
    }))
    .on('error', function(error) {
      console.error('Error uploading file:', error);
    })
    .on('finish', function() {
      console.log('File uploaded successfully');
    });
uploadFile().catch(console.error);

}