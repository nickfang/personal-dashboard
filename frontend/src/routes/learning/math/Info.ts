interface Shape {
  name: string;
  description: string;
  equations: {
    [key: string]: string; // Flexible equation keys
  };
}

type ShapesType = {
  [key: string]: Shape; // Flexible shape names
};

const shapes = {
  arc: {
    name: 'Arc',
    description: 'An arc is a portion of the circumference of a circle.',
    equations: {
      length: '2πr * (θ/360)',
      area: 'πr² * (θ/360)',
    },
  },
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

const terms = {
  point:
    'A point is a location in space that has no size or shape. In other words, it is a single location that is defined by its coordinates.',
  line: 'A line is a straight path that extends infinitely in both directions. In other words, it is a path that is defined by two points.',
  radius:
    'A radius is a line segment that connects the center of a circle to a point on the circumference of the circle. In other words, it is a segment of the circle that is defined by the center of the circle and a point on the circle.',
  diameter:
    'A diameter is a line segment that connects two points on the circumference of a circle and passes through the center of the circle. In other words, it is a segment of the circle that is defined by two points on the circle and the center of the circle.',
  circles:
    'A circle is a shape that is formed by all the points that are equidistant from a given point. In other words, it is a shape that is formed by the set of all points that are the same distance from a center point.',
  angle:
    'An angle is the space between two intersecting lines or surfaces. In other words, it is the space that is formed by two lines or surfaces that meet at a point.',
  'adjacent angles':
    'Two angles are said to be adjacent if they have a common vertex and a common side between them. In other words, they share a vertex and a side, but do not overlap or intersect.',
  'complementary angles':
    'Two angles are said to be complementary if the sum of their measures is equal to 90 degrees. In other words, if you add the measures of two complementary angles together, you will get 90 degrees.',
  'supplementary angles':
    'Two angles are said to be supplementary if the sum of their measures is equal to 180 degrees. In other words, if you add the measures of two supplementary angles together, you will get 180 degrees.',
  'vertical angles':
    'Two angles are said to be vertical angles if they are formed by two intersecting lines. In other words, they are opposite angles that are formed by the intersection of two lines.',
  'congruent angles':
    'Two angles are said to be congruent if they have the same measure. In other words, if you compare the measures of two congruent angles, you will find that they are equal.',
  'corresponding angles':
    'Two angles are said to be corresponding angles if they are in the same position in relation to two parallel lines and a transversal. In other words, they are angles that are in the same relative position in relation to two parallel lines and a transversal.',
  'interior angles':
    'An interior angle is an angle that is formed inside a polygon by two adjacent sides. In other words, it is an angle that is formed by two sides of a polygon that meet at a vertex inside the polygon.',
  'exterior angles':
    'An exterior angle is an angle that is formed outside a polygon by extending one of its sides. In other words, it is an angle that is formed by extending one of the sides of a polygon beyond the vertex.',
  'central angles':
    'A central angle is an angle that is formed at the center of a circle by two radii. In other words, it is an angle that is formed by two radii of a circle that meet at the center of the circle.',
  'inscribed angles':
    'An inscribed angle is an angle that is formed inside a circle by two chords. In other words, it is an angle that is formed by two chords of a circle that meet at a point on the circle.',
  ray: 'A ray is a straight path that extends infinitely in one direction. In other words, it is a path that is defined by a starting point and a direction.',
  arc: 'An arc is a portion of the circumference of a circle. In other words, it is a segment of the circle that is defined by two points on the circle.',
  chord:
    'A chord is a line segment that connects two points on the circumference of a circle. In other words, it is a segment of the circle that is defined by two points on the circle.',
  sector:
    'A sector is a portion of the area of a circle that is defined by two radii and an arc. In other words, it is a segment of the circle that is defined by two radii and an arc of the circle.',
  segment:
    'A segment is a portion of the area of a circle that is defined by a chord and an arc. In other words, it is a segment of the circle that is defined by a chord and an arc of the circle.',
  'congruent figures':
    'Two figures are said to be congruent if they have the same shape and size. In other words, if you compare two congruent figures, you will find that they are identical in shape and size.',
  'similar figures':
    'Two figures are said to be similar if they have the same shape but different sizes. In other words, if you compare two similar figures, you will find that they are identical in shape but not in size.',
  polygons:
    'A polygon is a closed figure that is made up of line segments. In other words, it is a shape that is formed by connecting line segments together to form a closed figure.',
  quadrilaterals:
    'A quadrilateral is a polygon that has four sides. In other words, it is a shape that is formed by connecting four line segments together to form a closed figure.',
  triangles:
    'A triangle is a polygon that has three sides. In other words, it is a shape that is formed by connecting three line segments together to form a closed figure.',
  ellipses:
    'An ellipse is a shape that is formed by the set of all points that are the same distance from two fixed points. In other words, it is a shape that is formed by the set of all points that are equidistant from two foci.',
  cones:
    'A cone is a three-dimensional shape that has a circular base and a curved surface that tapers to a point. In other words, it is a shape that is formed by a circle that is connected to a point by a curved surface.',
  cylinders:
    'A cylinder is a three-dimensional shape that has two circular bases that are connected by a curved surface. In other words, it is a shape that is formed by two circles that are connected by a curved surface.',
  spheres:
    'A sphere is a three-dimensional shape that is formed by all the points that are the same distance from a given point. In other words, it is a shape that is formed by the set of all points that are equidistant from a center point.',
  pyramids:
    'A pyramid is a three-dimensional shape that has a polygonal base and triangular faces that meet at a point. In other words, it is a shape that is formed by a polygon that is connected to a point by triangular faces.',
  prisms:
    'A prism is a three-dimensional shape that has two parallel bases that are connected by rectangular faces. In other words, it is a shape that is formed by two polygons that are connected by rectangular faces.',
  tori: 'A torus is a three-dimensional shape that is formed by a circle that is rotated around an axis. In other words, it is a shape that is formed by a circle that is rotated around a line.',
};

export { shapes, terms };
export type { ShapesType };
