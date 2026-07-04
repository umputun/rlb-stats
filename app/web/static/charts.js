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

        var theme = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : null;
        echarts.init(container, theme).setOption(data);
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

// dispose ECharts instances inside a node about to be replaced by an HTMX swap.
// hx-swap="innerHTML" removes the old chart containers from the DOM, so without this
// their instances leak across repeated period switches. only dispose when the swap
// will actually happen (shouldSwap is false for error responses / cancelled swaps),
// otherwise a failed period switch would blank the still-visible charts.
document.addEventListener("htmx:beforeSwap", function (evt) {
    if (!evt.detail || evt.detail.shouldSwap !== true || !evt.detail.target) return;
    evt.detail.target.querySelectorAll("[data-echarts]").forEach(function (container) {
        var instance = echarts.getInstanceByDom(container);
        if (instance) {
            instance.dispose();
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
