// Prevent dark mode flash - must be loaded in <head> (before body renders)
(function() {
    var savedTheme = localStorage.getItem('theme');
    var systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (savedTheme === 'dark' || (!savedTheme && systemPrefersDark)) {
        document.documentElement.classList.add('dark');
    }
})();
