let charts = {};
export function drawChart({ data, container, title }) {
  container.className = `chart echart`;
  const dataType = data.length > 0 && data[0].file ? "file" : "node";
  if (!charts[dataType]) {
    charts[dataType] = echarts.init(container);
  }
  // specify chart configuration item and data
  var option = {
    title: {
      text: title
    },
    grid: {
      height: "40%",
      width: "100%",
      bottom: "50%"
    },
    tooltip: {},
    legend: {
      type: "scroll",
      data: (function() {
        return data.map(datum => datum[dataType]);
      })(),
      bottom: 0,
      orient: "vertical",
      height: "40%"
    },
    xAxis: { type: "time" },
    yAxis: {},
    series: data.map(datum => {
      return {
        name: datum[dataType],
        encode: {
          x: 0,
          y: 1
        },
        type: "line",
        smooth: true,
        data: datum.data.map(el => [el[0], el[1]])
      };
    })
  };

  // use configuration item and data specified to show chart
  charts[dataType].setOption(option);
}

export function redraw() {
  charts.file.resize();
  charts.node.resize();
}
