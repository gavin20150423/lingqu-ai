try {
    var themeStore = JSON.parse(localStorage.getItem("infinite-canvas:theme_store") || "{}");
    var theme = themeStore.state && themeStore.state.theme === "light" ? "light" : "dark";
    document.documentElement.classList.toggle("dark", theme === "dark");
    document.documentElement.style.colorScheme = theme;
} catch (error) {
    // The application store applies the selected theme after startup.
}
