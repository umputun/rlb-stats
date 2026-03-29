// initCharts finds all [data-echarts] containers and initialises ECharts instances
// from JSON data in their sibling <script type="application/json"> elements.
function initCharts() {
    document.querySelectorAll("[data-echarts]").forEach(function (container) {
        var jsonScript = container.nextElementSibling;
        if (!jsonScript || jsonScript.type !== "application/json") return;

        var data;
        try {
            data = JSON.parse(jsonScript.textContent);
        } catch (e) {
            return;
        }

        // dispose existing instance before re-init (handles HTMX swaps)
        var existing = echarts.getInstanceByDom(container);
        if (existing) {
            existing.dispose();
        }

        echarts.init(container).setOption(data);
    });
}

// resize all active chart instances when window size changes
window.addEventListener("resize", function () {
    document.querySelectorAll("[data-echarts]").forEach(function (container) {
        var instance = echarts.getInstanceByDom(container);
        if (instance) {
            instance.resize();
        }
    });
});

// re-init charts after HTMX swaps new content
document.addEventListener("htmx:afterSettle", function () {
    initCharts();
});

// init charts on initial page load
document.addEventListener("DOMContentLoaded", function () {
    initCharts();
});
