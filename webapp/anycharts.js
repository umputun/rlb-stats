import { dateTimeOptions } from "./data.js";

function drawChart({ data, container, title }) {
  container.className = `chart anychart`;
  //clear the container
  container.innerHTML = `<h4>${title}</h4>`;
  const chart = anychart.line();
  data.map(elem => {
    const line = chart.spline(elem.data);
    if (elem.file) {
      line.name(elem.file);
    } else if (elem.node) {
      line.name(elem.node);
    }
    return line;
  });
  // set the container element
  chart.container(container);

  // enable the legend
  chart.legend(true);
  // set the layout of the legend
  chart.legend().itemsLayout("vertical-expandable");
  // set the position of the legend
  chart.legend().position("bottom");
  // set the maximum height and width of the legend
  chart.legend().height("220px");
  chart.height("500px");

  // enable html for the legend
  chart.legend().useHtml(true);

  // configure the format of legend items
  chart
    .legend()
    .itemsFormat(
      "<span style='color:#455a64;font-weight:600'>{%seriesName}:</span> {%seriesYSum}"
    );

  // configure the legend paginator
  const paginator = chart.legend().paginator();
  paginator.layout("vertical");
  paginator.fontSize(9);

  const labels = chart.xAxis().labels();
  labels.hAlign("center");
  labels.width(60);
  labels.fontSize(10);
  const xAxisLabels = chart.xAxis().labels();
  xAxisLabels.rotation(90);

  // create custom Date Time scale
  const dateTimeScale = anychart.scales.dateTime();
  const dateTimeTicks = dateTimeScale.ticks();

  // get period of time from the first and the last point in hours
  let periodInHours = 1;
  if (data.length > 0) {
    periodInHours =
      (data[0].data[data[0].data.length - 1][0] - data[0].data[0][0]) /
      (1000 * 60 * 60);
  }
  // format the x axis ticks
  switch (true) {
    case periodInHours <= 1:
      dateTimeTicks.interval("minutes", 5);
      break;
    case periodInHours <= 6:
      dateTimeTicks.interval("minutes", 20);
      break;
    case periodInHours <= 12:
      dateTimeTicks.interval("hours", 1);
      break;
    case periodInHours <= 24:
      dateTimeTicks.interval("hours", 2);
      break;
    case periodInHours <= 240:
      dateTimeTicks.interval("days", 1);
      break;
    case periodInHours <= 720:
      dateTimeTicks.interval("days", 3);
      break;
  }

  // apply Date Time scale
  chart.xScale(dateTimeScale);

  // set the alignment of the legend
  chart.legend().align("left");

  // tooltip settings
  const tooltip = chart.tooltip();
  tooltip.titleFormat(data => {
    return new Date(parseInt(data.kc.values.x.value)).toLocaleDateString(
      [],
      dateTimeOptions
    );
  });

  // initiate chart display
  chart.draw();
}

export { drawChart };
