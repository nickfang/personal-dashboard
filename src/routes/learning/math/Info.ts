const Lines = {
  Arc: {
    name: 'Arc',
    description: 'An arc is a portion of the circumference of a circle.',
    equations: {
      length: '2πr * (θ/360)',
      area: 'πr² * (θ/360)',
    },
  },
};

const Shapes = {
  circle: {
    name: 'Circle',
    description: 'A circle is a shape with all points the same distance from its center.',
    equations: {
      area: 'πr²',
      circumference: '2πr',
    },
  },
  triangle: {
    name: 'Triangle',
    description: 'A triangle is a polygon with three edges and three vertices.',
    equations: {
      area: '1/2 * base * height',
      perimeter: 'a + b + c',
    },
  },
  square: {
    name: 'Square',
    description:
      'A square is a regular quadrilateral, which means that it has four equal sides and four equal angles.',
    equations: {
      area: 'side * side',
      perimeter: '4 * side',
    },
  },
  rectangle: {
    name: 'Rectangle',
    description: 'A rectangle is a quadrilateral with four right angles.',
    equations: {
      area: 'length * width',
      perimeter: '2 * (length + width)',
    },
  },
  parallelogram: {
    name: 'Parallelogram',
    description: 'A parallelogram is a quadrilateral with opposite sides parallel.',
    equations: {
      area: 'base * height',
      perimeter: '2 * (side₁ + side₂)',
    },
  },
  rhombus: {
    name: 'Rhombus',
    description: 'A rhombus is a quadrilateral with all sides of equal length.',
    equations: {
      area: 'base * height',
      perimeter: '4 * side',
    },
  },
  trapezoid: {
    name: 'Trapezoid',
    description: 'A trapezoid is a quadrilateral with at least one pair of parallel sides.',
    equations: {
      area: '1/2 * (base₁ + base₂) * height',
      perimeter: 'side₁ + side₂ + base₁ + base₂',
    },
  },
  pentagon: {
    name: 'Pentagon',
    description: 'A pentagon is a five-sided polygon.',
    equations: {
      area: '1/4 * √(5 * (5 + 2 * √5)) * side²',
      perimeter: '5 * side',
    },
  },
  hexagon: {
    name: 'Hexagon',
    description: 'A hexagon is a six-sided polygon.',
    equations: {
      area: '3/2 * √3 * side²',
      perimeter: '6 * side',
    },
  },
  heptagon: {
    name: 'Heptagon',
    description: 'A heptagon is a seven-sided polygon.',
    equations: {
      area: '7/4 * side² * cot(π/7)',
      perimeter: '7 * side',
    },
  },
  octagon: {
    name: 'Octagon',
    description: 'An octagon is an eight-sided polygon.',
    equations: {
      area: '2 * side² * (1 + √2)',
      perimeter: '8 * side',
    },
  },
  nonagon: {
    name: 'Nonagon',
    description: 'A nonagon is a nine-sided polygon.',
    equations: {
      area: '9/4 * side² * cot(π/9)',
      perimeter: '9 * side',
    },
  },
  decagon: {
    name: 'Decagon',
    description: 'A decagon is a ten-sided polygon.',
    equations: {
      area: '5/2 * side² * √(5 + 2 * √5)',
      perimeter: '10 * side',
    },
  },
  dodecagon: {
    name: 'Dodecagon',
    description: 'A dodecagon is a twelve-sided polygon.',
    equations: {
      area: '3 * side² * √3 * (2 + √3)',
      perimeter: '12 * side',
    },
  },
};

const Volumes = {
  sphere: {
    name: 'Sphere',
    description: 'A sphere is a perfectly round geometrical object in three-dimensional space.',
    equations: {
      volume: '4/3 * π * r³',
      surfaceArea: '4 * π * r²',
    },
  },
  cone: {
    name: 'Cone',
    description:
      'A cone is a three-dimensional geometric shape that tapers smoothly from a flat base to a point called the apex or vertex.',
    equations: {
      volume: '1/3 * π * r² * h',
      surfaceArea: 'π * r * (r + l)',
    },
  },
  cube: {
    name: 'Cube',
    description:
      'A cube is a three-dimensional solid object bounded by six square faces, facets or sides, with three meeting at each vertex.',
    equations: {
      volume: 'side³',
      surfaceArea: '6 * side²',
    },
  },
  cylinder: {
    name: 'Cylinder',
    description:
      'A cylinder is one of the most basic curvilinear geometric shapes, the surface formed by the points at a fixed distance from a given line segment, the axis of the cylinder.',
    equations: {
      volume: 'π * r² * h',
      surfaceArea: '2 * π * r * (r + h)',
    },
  },
  pyramid: {
    name: 'Pyramid',
    description:
      'A pyramid is a polyhedron formed by connecting a polygonal base and a point, called the apex.',
    equations: {
      volume: '1/3 * base * height',
      surfaceArea: 'base + 1/2 * perimeter * slant height',
    },
  },
};

export { Lines, Shapes, Volumes };
