//DOMContentLoaded starts when the HTML has fully loaded 
document.addEventListener('DOMContentLoaded', function() {
    const filterForm = document.querySelector('.filter-form');
    const filterInputs = filterForm.querySelectorAll('select, input[type="checkbox"]');

    //every filter input has an event listener that listens for a change event
    filterInputs.forEach(input => {
        input.addEventListener('change', () => {
            filterForm.submit(); //if the user changes the value of a filter, the form is submitted
        });
    });

    //creating a reset button
    const resetButton = document.createElement('button');
    resetButton.type = 'button';
    resetButton.textContent = 'Reset Filters';
    resetButton.className = 'reset-filters';
    resetButton.onclick = () => {
        //resetting the value of the select element and unchecking all checkboxes
        filterForm.querySelector('select').value = '';
        filterForm.querySelectorAll('input[type="checkbox"]').forEach(checkbox => {
            checkbox.checked = false;
        });
        //sending the form to load the page with the default filters
        filterForm.submit();
    };

    //adding the reset button after the submit button
    filterForm.querySelector('button[type="submit"]').after(resetButton);
}); 