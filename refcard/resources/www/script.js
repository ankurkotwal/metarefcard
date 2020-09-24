$(function(){
    let inputFile = $('#filesInput');
    let addButton = $('#addButton');
    let generateButton = $('#generateButton');
    let filesContainer = $('#files');
    let files = [];
    
    inputFile.change(function() {
      let newFiles = []; 
      for(let index = 0; index < inputFile[0].files.length; index++) {
        let file = inputFile[0].files[index];
        newFiles.push(file);
        files.push(file);
      }
      
      newFiles.forEach(file => {
        let fileElement = $(`<option>${file.name}</option>`);
        fileElement.data('fileData', file);
        filesContainer.append(fileElement);
        
        fileElement.click(function(event) {
          let fileElement = $(event.target);
          let indexToRemove = files.indexOf(fileElement.data('fileData'));
          fileElement.remove();
          files.splice(indexToRemove, 1);
        });
      });
    });
    
    addButton.click(function() {
      inputFile.click();
    });
    
    generateButton.click(function() {
      let formData = new FormData();
      
      files.forEach(file => {
        formData.append('file', file);
      });
      
      console.log('Sending...');
      
      $.ajax({
        url: '/fs2020',
        data: formData,
        type: 'POST',
        success: function(data) {
          imageContainer = $('#images')
          imageContainer.empty();
          var b64Data = btoa(unescape(encodeURIComponent(data)));
          var outputImg = document.createElement('img')
          outputImg.src = 'data:image/jpg;base64,' + b64Data;
          imageContainer.html(outputImg);
          // imageContainer.append('<p>SUCCESS !!!</p>'); 
        },
        error: function(data) { console.log('ERROR !!!'); },
        cache: false,
        processData: false,
        contentType: false
      });
    });
  });