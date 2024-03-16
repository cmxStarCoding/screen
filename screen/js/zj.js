


window.onload = function() {

    setTimeout(function (){
        var element = document.querySelector(".ovui-modal__wrap .ovui-modal .ovui-modal__close-icon");
        if (element) {
            element.click();
        }
    },1500)
    setTimeout(function (){
        var element = document.querySelector(".ovui-button.ovui-button--md.ovui-button--rect.ovui-button--primary.ovui-button--primary-fill.ovui-button--fill");
        if (element) {
            element.click()
        }
    },2000)

    setTimeout(function (){
        var elements = document.querySelectorAll(".ant-tabs-tab");
        if (elements.length >= 2) {
            elements[2].click();
        }
        //document.getElementById("adv").value = "1788866784576523";
    },2500)

    setTimeout(function (){
        document.getElementById("adv").value = "1788866784576523";
    },8000)

    setTimeout(function (){
        var buttons = document.querySelectorAll('button[ant-click-animating-without-extra-node="false"]');
        buttons.forEach(function(button) {
            button.click();
        });
    },8000)


}



