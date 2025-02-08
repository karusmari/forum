// //DOMContentLoaded starts when the HTML has fully loaded 
// document.addEventListener('DOMContentLoaded', function() {
//     const filterForm = document.querySelector('.filter-form');


//     // const filterInputs = filterForm.querySelectorAll('select, input[type="checkbox"]');

//     // //every filter input has an event listener that listens for a change event
//     // filterInputs.forEach(input => {
//     //     input.addEventListener('change', () => {
//     //         filterForm.submit(); //if the user changes the value of a filter, the form is submitted
//     //     });
//     // });

//     //creating a reset button
//     const resetButton = document.createElement('button');
//     resetButton.type = 'button';
//     resetButton.textContent = 'Reset Filters';
//     resetButton.className = 'reset-filters';
//     resetButton.onclick = () => {
//         //resetting the value of the select element and unchecking all checkboxes
//         filterForm.querySelector('select').value = '';
//         filterForm.querySelectorAll('input[type="checkbox"]').forEach(checkbox => {
//             checkbox.checked = false;
//         });
//         //sending the form to load the page with the default filters
//         filterForm.submit();
//     };

//     //adding the reset button after the submit button
//     filterForm.querySelector('button[type="submit"]').after(resetButton);
// }); 

function filterPosts(filter) {
    const posts = document.querySelectorAll('.post-preview');
    
    posts.forEach(post => {
        let show = false;
        switch(filter) {
            case 'my':
                show = post.dataset.isMine === "true";
                break;
            case 'liked':
                show = post.dataset.isLiked === "true";
                break;
            default: // 'all'
                show = true;
                break;
        }
        
        post.style.display = show ? 'block' : 'none';
        if (show) {
            post.style.opacity = '1';
        } else {
            post.style.opacity = '0';
        }
    });

    // update active button
    document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    event.target.classList.add('active');
}