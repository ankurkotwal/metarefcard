$(function () {
  let inputFile = $('#filesInput');
  let addButton = $('#addButton');
  let generateButton = $('#generateButton');
  let filesContainer = $('#files');
  let files = [];

  inputFile.change(function () {
    generateButton.attr("disabled", true);
    let newFiles = [];
    for (let index = 0; index < inputFile[0].files.length; index++) {
      generateButton.attr("disabled", false);
      let file = inputFile[0].files[index];
      newFiles.push(file);
      files.push(file);
    }

    newFiles.forEach(file => {
      let fileElement = $(`<option>${file.name}</option>`);
      fileElement.data('fileData', file);
      filesContainer.append(fileElement);

      fileElement.click(function (event) {
        let fileElement = $(event.target);
        let indexToRemove = files.indexOf(fileElement.data('fileData'));
        fileElement.remove();
        files.splice(indexToRemove, 1);

        if (filesContainer.children().length == 0) {
          generateButton.attr("disabled", true);
        } else {
          generateButton.attr("disabled", false);
        }
      });
    });
    $(this).val('') // Makes it possible to add, remove, add same file
  });

  addButton.click(function () {
    inputFile.click();
  });

  generateButton.click(function () {
    $('#images').empty();
    $('#progressbar').show();
    let formData = new FormData();

    files.forEach(file => {
      formData.append('file', file);
    });

    $.ajax({
      url: '/fs2020',
      data: formData,
      type: 'POST',
      success: function (data) {
        $('#progressbar').hide();
        imageContainer = $('#images');
        imageContainer.empty();
        imageContainer.html(data);
      },
      error: function (data) {
        $('#progressbar').hide();
        console.log('ERROR !!!');
      },
      cache: false,
      processData: false,
      contentType: false
    });
  });
});