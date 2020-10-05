// A minimal polyfill for `navigator.clipboard.writeText()` that works most of the time in most modern browsers.
// Note that on Edge this may call `resolve()` even if copying failed.
// See https://github.com/lgarron/clipboard-polyfill for a more robust solution.
// License: public domain
function copyTextFromElement(element) {
  return new Promise(function (resolve, reject) {

    /********************************/
    var range = document.createRange();
    range.selectNodeContents(document.body);
    document.getSelection().addRange(range);
    /********************************/

    var success = false;
    function listener(e) {
      e.clipboardData.setData("text/plain", element.innerHTML);
      e.preventDefault();
      success = true;
    }
    document.addEventListener("copy", listener);
    document.execCommand("copy");
    document.removeEventListener("copy", listener);

    /********************************/
    document.getSelection().removeAllRanges();
    /********************************/

    success ? resolve() : reject();
  });
};

$(function () {
  let files = [];

  // ##### Flight Simulator 2020 #####
  let fs2020InputFile = $('#fs2020FilesInput');
  let fs2020AddButton = $('#fs2020AddButton');
  let fs2020GenerateButton = $('#fs2020GenerateButton');
  let fs2020FilesContainer = $('#fs2020Files');
  fs2020InputFile.change(function () {
    inputFileChange(fs2020GenerateButton, fs2020InputFile, files, fs2020FilesContainer);
    $(this).val('') // Makes it possible to add, remove, add same file
  });
  fs2020AddButton.click(function () {
    fs2020InputFile.click();
  });
  fs2020GenerateButton.click(function () {
    callBackend('/fs2020', files, $('#fs2020Progressbar'), $('#fs2020Images'))
  });
});

function inputFileChange(generateButton, inputFile, files, filesContainer) {
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
}

function callBackend(url, files, progressbar, imageContainer) {
  let formData = new FormData();
  files.forEach(file => {
    formData.append('file', file);
  });

  imageContainer.empty();
  progressbar.show();
  $.ajax({
    url: url,
    data: formData,
    type: 'POST',
    success: function (data) {
      progressbar.hide();
      imageContainer.html(data);
    },
    error: function (data) {
      progressbar.hide();
      console.log('ERROR !!!');
    },
    cache: false,
    processData: false,
    contentType: false
  });
}