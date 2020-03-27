export function drawChart({data, container, title}) {
  // TODO this fails in case of empty data
  const dataType = data[0].file ? "file" : "node";
  const myChart = echarts.init(container);
  // specify chart configuration item and data
  var option = {
    title: {
      text: title
    },
    tooltip: {},
    legend: {
      type: 'scroll',
      data: (function () {
        return data.map((datum) => datum[dataType]);
      })()
    },
    xAxis: {type: "time"},
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
  myChart.setOption(option);
}
